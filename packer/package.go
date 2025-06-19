package packer

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

const PackageYaml = "package.yaml"

// Package represents a Packer package
type Package struct {
	Name            string `yaml:"name"`   // user/repo
	Source          string `yaml:"source"` // source repository
	CommitHashShort string `yaml:"commit"` // commit hash
}

// PackageFile (package.yaml)
type PackageFile struct {
	Author   string     `yaml:"author"`
	Packages []*Package `yaml:"packages"`
}

func LoadPackageFile(root string) (*PackageFile, error) {
	path := filepath.Join(root, PackageYaml)

	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &PackageFile{Author: "Unknown"}, nil
		}
		return nil, err
	}
	defer file.Close()

	var pf PackageFile
	if err := yaml.NewDecoder(file).Decode(&pf); err != nil {
		return nil, err
	}

	return &pf, nil
}

func (pf *PackageFile) Save(root string) error {
	path := filepath.Join(root, PackageYaml)
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	return yaml.NewEncoder(file).Encode(pf)
}
