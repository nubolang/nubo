package config

import (
	"os"
	"path/filepath"
)

func BaseDir() (string, error) {
	base, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	dir := filepath.Join(base, "nubo")
	if err := os.MkdirAll(dir, os.ModePerm); err != nil {
		return "", err
	}
	return dir, nil
}
