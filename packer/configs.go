package packer

import (
	"os"
	"path/filepath"

	"go.uber.org/zap"
)

func BaseDir() (string, error) {
	base, err := os.UserHomeDir()
	if err != nil {
		zap.L().Error("packer.baseDir.homeDir", zap.Error(err))
		return "", err
	}
	return makeDir(filepath.Join(base, "nubo"))
}

func PackageDir() (string, error) {
	base, err := BaseDir()
	if err != nil {
		zap.L().Error("packer.packageDir.base", zap.Error(err))
		return "", err
	}

	return makeDir(filepath.Join(base, "packages"))
}

func makeDir(dir string) (string, error) {
	if err := os.MkdirAll(dir, os.ModePerm); err != nil {
		zap.L().Error("packer.makeDir", zap.String("dir", dir), zap.Error(err))
		return "", err
	}
	zap.L().Debug("packer.makeDir.ok", zap.String("dir", dir))
	return dir, nil
}
