package packer

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"

	"github.com/manifoldco/promptui"
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
	Name       string     `yaml:"name"`
	Repository string     `yaml:"repository,omitempty"`
	Packages   []*Package `yaml:"packages"`
}

func LoadPackageFile(root string, forceCreate bool) (*PackageFile, error) {
	path := filepath.Join(root, PackageYaml)

	file, err := os.Open(path)
	if err != nil {
		if !forceCreate {
			return &PackageFile{}, nil
		}
		if os.IsNotExist(err) {
			return getPackage()
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

func getPackage() (*PackageFile, error) {
	validate := func(input string) error {
		matched, _ := regexp.MatchString(`^[a-zA-Z0-9_.-]+$`, input)
		if !matched {
			return fmt.Errorf("must contain only letters, numbers, _, -, .")
		}
		return nil
	}

	validateURL := func(input string) error {
		if input == "" {
			return nil
		}
		_, err := url.ParseRequestURI(input)
		if err != nil {
			return fmt.Errorf("must be a valid URL")
		}
		return nil
	}

	authorPrompt := promptui.Prompt{
		Label:    "Init: Author",
		Validate: validate,
	}
	author, err := authorPrompt.Run()
	if err != nil {
		return nil, err
	}

	projectPrompt := promptui.Prompt{
		Label:    "Init: Project",
		Validate: validate,
	}
	project, err := projectPrompt.Run()
	if err != nil {
		return nil, err
	}

	repoPrompt := promptui.Prompt{
		Label:    "Init: Repository",
		Validate: validateURL,
	}
	repository, err := repoPrompt.Run()
	if err != nil {
		return nil, err
	}

	return &PackageFile{
		Name:       author + ":" + project,
		Repository: repository,
	}, nil
}

func (pf *PackageFile) Find(url string) (*Package, error) {
	for _, p := range pf.Packages {
		if p.Source == url {
			return p, nil
		}
	}
	return nil, fmt.Errorf("package not found")
}
