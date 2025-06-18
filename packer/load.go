package packer

import (
	"os"

	"gopkg.in/yaml.v3"
)

func (p *Packer) Load(lock string, cachePath string) ([]*LockEntry, error) {
	file, err := os.Open(lock)
	if err != nil {
		return nil, err
	}

	var lockFile *LockFile
	if err := yaml.NewDecoder(file).Decode(&lockFile); err != nil {
		return nil, err
	}

	for _, entry := range lockFile.Entries {
		_, err := entry.Download(cachePath)
		if err != nil {
			return nil, err
		}
	}

	return lockFile.Entries, nil
}
