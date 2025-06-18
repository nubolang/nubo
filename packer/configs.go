package packer

import (
	"os"
	"path/filepath"
)

func BaseDir() (string, error) {
	base, err := os.UserCacheDir()
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

func CacheDir() (string, error) {
	base, err := BaseDir()
	if err != nil {
		return "", err
	}

	return makeDir(filepath.Join(base, "cache"))
}

func SumDBDir() (string, error) {
	base, err := BaseDir()
	if err != nil {
		return "", err
	}

	return makeDir(filepath.Join(base, "sumdb"))
}

func makeDir(dir string) (string, error) {
	if err := os.MkdirAll(dir, os.ModePerm); err != nil {
		return "", err
	}
	return dir, nil
}
