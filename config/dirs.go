package config

import (
	"errors"
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

func GetFile() (string, error) {
	dir, err := BaseDir()
	if err != nil {
		return "", err
	}

	file := filepath.Join(dir, "config.yaml")
	if _, err := os.Stat(file); os.IsNotExist(err) {
		return "", errors.New("config.yaml does not exists")
	}
	return file, nil
}
