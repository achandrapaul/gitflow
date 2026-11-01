package prefs_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/achandrapaul/gitflow/internal/prefs"
)

func TestStoreLoad(t *testing.T) {
	testCases := []struct {
		name       string
		content    string
		wantLayout prefs.Layout
		wantErr    bool
	}{
		{name: "first run defaults to flat", wantLayout: prefs.LayoutFlat},
		{name: "saved folder", content: `{"layout":"folder"}`, wantLayout: prefs.LayoutFolder},
		{name: "saved flat", content: `{"layout":"flat"}`, wantLayout: prefs.LayoutFlat},
		{name: "corrupt file falls back to flat", content: `{"layout":`, wantLayout: prefs.LayoutFlat, wantErr: true},
		{name: "unknown layout falls back to flat", content: `{"layout":"tree"}`, wantLayout: prefs.LayoutFlat, wantErr: true},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "gitflow", "prefs.json")
			if testCase.content != "" {
				if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(path, []byte(testCase.content), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			preferences, err := prefs.NewStore(path).Load()
			if (err != nil) != testCase.wantErr || preferences.Layout != testCase.wantLayout {
				t.Fatalf("Load = %+v, %v; want layout %v, error %v", preferences, err, testCase.wantLayout, testCase.wantErr)
			}
		})
	}
}

func TestStoreSaveSurvivesRestart(t *testing.T) {
	directory := t.TempDir()
	path := filepath.Join(directory, "gitflow", "prefs.json")
	for _, layout := range []prefs.Layout{prefs.LayoutFolder, prefs.LayoutFlat, prefs.LayoutFolder} {
		if err := prefs.NewStore(path).Save(prefs.Preferences{Layout: layout}); err != nil {
			t.Fatalf("Save(%v): %v", layout, err)
		}
		preferences, err := prefs.NewStore(path).Load()
		if err != nil || preferences.Layout != layout {
			t.Fatalf("after restart Load = %+v, %v; want %v", preferences, err, layout)
		}
	}
	entries, err := os.ReadDir(filepath.Dir(path))
	if err != nil || len(entries) != 1 {
		t.Fatalf("config directory holds %v, %v; want only prefs.json", entries, err)
	}
}

func TestStoreUnavailableLocation(t *testing.T) {
	blocker := filepath.Join(t.TempDir(), "not-a-directory")
	if err := os.WriteFile(blocker, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	store := prefs.NewStore(filepath.Join(blocker, "prefs.json"))
	if err := store.Save(prefs.Preferences{Layout: prefs.LayoutFolder}); err == nil {
		t.Fatal("Save into an unavailable location reported success")
	}
	preferences, err := store.Load()
	if err == nil || preferences.Layout != prefs.LayoutFlat {
		t.Fatalf("Load = %+v, %v; want flat with error", preferences, err)
	}
}
