package packer

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"path/filepath"
	"sort"
)

func hashDir(path string) (string, error) {
	hasher := sha256.New()
	var files []string

	err := filepath.Walk(path, func(p string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}
		files = append(files, p)
		return nil
	})
	if err != nil {
		return "", err
	}

	sort.Strings(files)

	for _, file := range files {
		f, err := os.Open(file)
		if err != nil {
			return "", err
		}
		defer f.Close()

		if _, err := io.Copy(hasher, f); err != nil {
			return "", err
		}
	}

	return hex.EncodeToString(hasher.Sum(nil)), nil
}
