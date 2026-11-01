package prefs

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

type Layout int

const (
	LayoutFlat Layout = iota
	LayoutFolder
)

const (
	flatLayoutName   = "flat"
	folderLayoutName = "folder"
	directoryMode    = 0o755
	tempFilePattern  = ".prefs-*.tmp"
)

var layoutNames = map[Layout]string{LayoutFlat: flatLayoutName, LayoutFolder: folderLayoutName}

type Preferences struct {
	Layout Layout
}

type storedPreferences struct {
	Layout string `json:"layout"`
}

type Store struct {
	path string
}

func NewStore(path string) *Store {
	return &Store{path: path}
}

func (store *Store) Load() (Preferences, error) {
	content, err := os.ReadFile(store.path)
	if errors.Is(err, fs.ErrNotExist) {
		return Preferences{Layout: LayoutFlat}, nil
	}
	if err != nil {
		return Preferences{Layout: LayoutFlat}, fmt.Errorf("read preferences %q: %w", store.path, err)
	}
	var stored storedPreferences
	if err := json.Unmarshal(content, &stored); err != nil {
		return Preferences{Layout: LayoutFlat}, fmt.Errorf("parse preferences %q: %w", store.path, err)
	}
	for layout, name := range layoutNames {
		if name == stored.Layout {
			return Preferences{Layout: layout}, nil
		}
	}
	return Preferences{Layout: LayoutFlat}, fmt.Errorf("parse preferences %q: unknown layout %q", store.path, stored.Layout)
}

func (store *Store) Save(preferences Preferences) error {
	content, err := json.Marshal(storedPreferences{Layout: layoutNames[preferences.Layout]})
	if err != nil {
		return fmt.Errorf("encode preferences: %w", err)
	}
	directory := filepath.Dir(store.path)
	if err := os.MkdirAll(directory, directoryMode); err != nil {
		return fmt.Errorf("create preferences directory %q: %w", directory, err)
	}
	tempFile, err := os.CreateTemp(directory, tempFilePattern)
	if err != nil {
		return fmt.Errorf("create temporary preferences file in %q: %w", directory, err)
	}
	defer os.Remove(tempFile.Name())
	if _, err := tempFile.Write(content); err != nil {
		tempFile.Close()
		return fmt.Errorf("write temporary preferences file %q: %w", tempFile.Name(), err)
	}
	if err := tempFile.Close(); err != nil {
		return fmt.Errorf("close temporary preferences file %q: %w", tempFile.Name(), err)
	}
	if err := os.Rename(tempFile.Name(), store.path); err != nil {
		return fmt.Errorf("replace preferences %q: %w", store.path, err)
	}
	return nil
}
