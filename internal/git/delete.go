package git

import (
	"context"
	"io/fs"
	"time"
)

type FileIdentity struct {
	Mode    fs.FileMode
	Size    int64
	ModTime time.Time
	Inode   uint64
}

type FileDeleter struct {
	runner *Runner
}

func NewFileDeleter(runner *Runner) *FileDeleter {
	return &FileDeleter{runner: runner}
}

func (d *FileDeleter) Identify(path string) (FileIdentity, error) {
	return FileIdentity{}, errNotImplemented
}

func (d *FileDeleter) Delete(ctx context.Context, path string, expected FileIdentity) error {
	return errNotImplemented
}
