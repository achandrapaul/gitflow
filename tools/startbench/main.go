package main

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"time"

	"github.com/creack/pty"
)

const (
	terminalRows         = 36
	terminalColumns      = 120
	frameTimeout         = 5 * time.Second
	fixtureCommitCount   = 500
	fixtureChangedFiles  = 40
	readChunkSize        = 4096
	fixtureEpoch         = 1700000000
	settleBetweenRuns    = 300 * time.Millisecond
	cursorPositionQuery  = "\x1b[6n"
	backgroundColorReply = "\x1b]11;rgb:0000/0000/0000\x1b\\"
	cursorPositionReply  = "\x1b[1;1R"
)

func main() {
	binaryPath := flag.String("bin", "", "path to the binary under test")
	repositoryDir := flag.String("repo", "", "repository to launch in (default: generated fixture)")
	frameMarker := flag.String("marker", "CHANGES", "text that marks the first interactive frame")
	runCount := flag.Int("runs", 21, "number of launches; the first is reported as cold")
	flag.Parse()

	fail := func(format string, arguments ...any) {
		fmt.Fprintf(os.Stderr, "startbench: "+format+"\n", arguments...)
		os.Exit(1)
	}
	if *binaryPath == "" || *runCount < 2 {
		fail("-bin is required and -runs must be at least 2")
	}
	absoluteBinary, err := filepath.Abs(*binaryPath)
	if err != nil {
		fail("resolve binary: %v", err)
	}

	launchDir := *repositoryDir
	if launchDir == "" {
		launchDir, err = os.MkdirTemp("", "startbench-fixture-")
		if err != nil {
			fail("create fixture dir: %v", err)
		}
		defer os.RemoveAll(launchDir)
		var fastImport strings.Builder
		for commitNumber := 1; commitNumber <= fixtureCommitCount; commitNumber++ {
			commitMessage := fmt.Sprintf("commit %d", commitNumber)
			fileContent := commitMessage + "\n"
			fmt.Fprintf(&fastImport, "commit refs/heads/main\ncommitter Bench <bench@example.invalid> %d +0000\ndata %d\n%s\n",
				fixtureEpoch+commitNumber, len(commitMessage), commitMessage)
			fmt.Fprintf(&fastImport, "M 100644 inline file-%d.txt\ndata %d\n%s\n", commitNumber%fixtureChangedFiles, len(fileContent), fileContent)
		}
		for _, arguments := range [][]string{{"init", "-q", "-b", "main"}, {"fast-import", "--quiet"}, {"checkout", "-q", "main"}} {
			command := exec.Command("git", arguments...)
			command.Dir = launchDir
			if arguments[0] == "fast-import" {
				command.Stdin = strings.NewReader(fastImport.String())
			}
			if output, err := command.CombinedOutput(); err != nil {
				fail("git %s: %v: %s", strings.Join(arguments, " "), err, output)
			}
		}
		for fileNumber := range fixtureChangedFiles {
			filePath := filepath.Join(launchDir, fmt.Sprintf("file-%d.txt", fileNumber))
			if err := os.WriteFile(filePath, []byte("changed in worktree\n"), 0o644); err != nil {
				fail("modify fixture file: %v", err)
			}
		}
	}

	durations := make([]time.Duration, 0, *runCount)
	for range *runCount {
		command := exec.Command(absoluteBinary)
		command.Dir = launchDir
		command.Env = append(os.Environ(), "TERM=xterm-256color")
		startedAt := time.Now()
		terminal, err := pty.StartWithSize(command, &pty.Winsize{Rows: terminalRows, Cols: terminalColumns})
		if err != nil {
			fail("start %s: %v", absoluteBinary, err)
		}
		frameSeen := make(chan time.Duration, 1)
		go func() {
			var output bytes.Buffer
			chunk := make([]byte, readChunkSize)
			answeredQuery := false
			for {
				readCount, readErr := terminal.Read(chunk)
				output.Write(chunk[:readCount])
				if !answeredQuery && strings.Contains(output.String(), cursorPositionQuery) {
					answeredQuery = true
					_, _ = terminal.WriteString(backgroundColorReply + cursorPositionReply)
				}
				if strings.Contains(output.String(), *frameMarker) {
					frameSeen <- time.Since(startedAt)
					return
				}
				if readErr != nil {
					return
				}
			}
		}()
		select {
		case elapsed := <-frameSeen:
			durations = append(durations, elapsed)
		case <-time.After(frameTimeout):
			fail("first frame marker %q not seen within %s", *frameMarker, frameTimeout)
		}
		_ = command.Process.Kill()
		_ = command.Wait()
		_ = terminal.Close()
		time.Sleep(settleBetweenRuns)
	}

	machine := runtime.GOOS + "/" + runtime.GOARCH
	if unameOutput, err := exec.Command("uname", "-sr").Output(); err == nil {
		machine += ", " + strings.TrimSpace(string(unameOutput))
	}
	if cpuOutput, err := exec.Command("sysctl", "-n", "machdep.cpu.brand_string").Output(); err == nil {
		machine += ", " + strings.TrimSpace(string(cpuOutput))
	}
	if cpuInfo, err := os.ReadFile("/proc/cpuinfo"); err == nil {
		for _, line := range strings.Split(string(cpuInfo), "\n") {
			if strings.HasPrefix(line, "model name") {
				machine += ", " + strings.TrimSpace(strings.SplitN(line, ":", 2)[1])
				break
			}
		}
	}

	warmDurations := slices.Clone(durations[1:])
	slices.Sort(warmDurations)
	fmt.Printf("machine: %s\n", machine)
	fmt.Printf("terminal: pty %dx%d, TERM=xterm-256color\n", terminalColumns, terminalRows)
	fmt.Printf("fixture: %s\n", launchDir)
	fmt.Printf("cold (first launch in fresh fixture; OS cache not purged; binary already executed once): %s\n", durations[0])
	fmt.Printf("warm runs: %d\n", len(warmDurations))
	fmt.Printf("warm p50: %s\n", warmDurations[len(warmDurations)*50/100])
	fmt.Printf("warm p95: %s\n", warmDurations[min(len(warmDurations)-1, len(warmDurations)*95/100)])
	fmt.Printf("warm max: %s\n", warmDurations[len(warmDurations)-1])
}
