package packer

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"path/filepath"
	"sort"

	"go.uber.org/zap"
)

func hashDir(path string) (string, error) {
	hasher := sha256.New()
	var files []string

	err := filepath.Walk(path, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() && info.Name() == ".git" {
			return filepath.SkipDir
		}
		if info.IsDir() {
			return nil
		}

		files = append(files, p)
		return nil
	})
	if err != nil {
		return "", err
	}

	sort.Strings(files)
	zap.L().Debug("packer.hashDir.walked", zap.String("path", path), zap.Int("fileCount", len(files)))

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

	sum := hex.EncodeToString(hasher.Sum(nil))
	zap.L().Debug("packer.hashDir.complete", zap.String("path", path), zap.String("hash", sum))
	return sum, nil
}
