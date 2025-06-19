package packer

import (
	"path/filepath"
	"strings"
)

func (p *Packer) ImportFile(path string) (string, error) {
	for _, entry := range p.Lock.Entries {
		if strings.HasPrefix(path, entry.Name) {
			remaining := strings.TrimPrefix(path, entry.Name)
			if remaining == "" {
				parts := strings.Split(entry.Name, "/")
				remaining = parts[len(parts)-1]
			}
			cache, err := PackageDir()
			if err != nil {
				return "", err
			}
			return filepath.Join(cache, entry.Domain(), entry.Name+"@"+entry.CommitHash, remaining), nil
		}
	}
	return path, nil
}
