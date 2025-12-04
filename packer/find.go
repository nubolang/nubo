package packer

import (
	"path/filepath"
	"strings"

	"go.uber.org/zap"
)

func (p *Packer) ImportFile(path string) (string, error) {
	zap.L().Debug("packer.importFile.start", zap.String("path", path))
	for _, entry := range p.Lock.Entries {
		if strings.HasPrefix(path, entry.Name) {
			remaining := strings.TrimPrefix(path, entry.Name)
			if remaining == "" {
				parts := strings.Split(entry.Name, "/")
				remaining = parts[len(parts)-1]
			}
			cache, err := PackageDir()
			if err != nil {
				zap.L().Error("packer.importFile.packageDir", zap.Error(err))
				return "", err
			}
			target := filepath.Join(cache, entry.Domain(), entry.Name+"@"+entry.CommitHash, remaining)
			zap.L().Debug("packer.importFile.mapped", zap.String("input", path), zap.String("output", target))
			return target, nil
		}
	}
	zap.L().Debug("packer.importFile.noMatch", zap.String("path", path))
	return path, nil
}
