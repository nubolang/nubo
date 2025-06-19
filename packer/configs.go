package packer

import (
	"os"
	"path/filepath"
)

func BaseDir() (string, error) {
	base, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return makeDir(filepath.Join(base, "nubo"))
}

func PackageDir() (string, error) {
	base, err := BaseDir()
	if err != nil {
		return "", err
	}

	return makeDir(filepath.Join(base, "packages"))
}

func makeDir(dir string) (string, error) {
	if err := os.MkdirAll(dir, os.ModePerm); err != nil {
		return "", err
	}
	return dir, nil
}
