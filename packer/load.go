package packer

import (
	"errors"
	"os"

	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

func (p *Packer) Load(lock string, cachePath string) ([]*LockEntry, error) {
	zap.L().Debug("packer.load.start", zap.String("path", lock))
	file, err := os.Open(lock)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			zap.L().Warn("package does not has a lock file", zap.String("lock", lock), zap.String("cachePath", cachePath))
			return make([]*LockEntry, 0), nil
		}

		zap.L().Error("packer.load.open", zap.String("path", lock), zap.Error(err))
		return nil, err
	}

	var lockFile *LockFile
	if err := yaml.NewDecoder(file).Decode(&lockFile); err != nil {
		zap.L().Error("packer.load.decode", zap.String("path", lock), zap.Error(err))
		return nil, err
	}

	for _, entry := range lockFile.Entries {
		_, _, err := entry.Download(cachePath)
		if err != nil {
			zap.L().Error("packer.load.download", zap.String("entry", entry.Name), zap.Error(err))
			return nil, err
		}
	}

	zap.L().Debug("packer.load.success", zap.Int("entries", len(lockFile.Entries)))
	return lockFile.Entries, nil
}
