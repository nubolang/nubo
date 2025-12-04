package packer

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"

	"github.com/manifoldco/promptui"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

const PackageYaml = "_nubo.yaml"

// Package represents a Packer package
type Package struct {
	Name            string `yaml:"-"`      // user/repo/etc
	Source          string `yaml:"source"` // source repository
	CommitHashShort string `yaml:"commit"` // commit hash
}

type PackageAuthor struct {
	Name    string `yaml:"name"`
	Website string `yaml:"website"`
}

// PackageFile (_nubo.yaml)
type PackageFile struct {
	Name       string        `yaml:"name"`
	Author     PackageAuthor `yaml:"author"`
	Repository string        `yaml:"repository,omitempty"`
	Packages   []*Package    `yaml:"packages"`
}

func LoadPackageFile(root string, forceCreate bool) (*PackageFile, error) {
	path := filepath.Join(root, PackageYaml)

	file, err := os.Open(path)
	if err != nil {
		if !forceCreate {
			zap.L().Info("packer.package.load.empty", zap.String("path", path))
			return &PackageFile{}, nil
		}
		if os.IsNotExist(err) {
			zap.L().Info("packer.package.load.new", zap.String("path", path))
			return getPackage()
		}
		zap.L().Error("packer.package.load.open", zap.String("path", path), zap.Error(err))
		return nil, err
	}
	defer file.Close()

	var pf PackageFile
	if err := yaml.NewDecoder(file).Decode(&pf); err != nil {
		zap.L().Error("packer.package.load.decode", zap.String("path", path), zap.Error(err))
		return nil, err
	}

	zap.L().Debug("packer.package.load.success", zap.Int("packages", len(pf.Packages)))
	return &pf, nil
}

func (pf *PackageFile) Save(root string) error {
	path := filepath.Join(root, PackageYaml)
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		zap.L().Error("packer.package.save.open", zap.String("path", path), zap.Error(err))
		return err
	}
	defer file.Close()

	if err := yaml.NewEncoder(file).Encode(pf); err != nil {
		zap.L().Error("packer.package.save.encode", zap.String("path", path), zap.Error(err))
		return err
	}

	zap.L().Debug("packer.package.save.success", zap.String("path", path), zap.Int("packages", len(pf.Packages)))
	return nil
}

func getPackage() (*PackageFile, error) {
	zap.L().Info("packer.package.get.start")
	validate := func(input string) error {
		matched, _ := regexp.MatchString(`^[a-zA-Z0-9_.-]+$`, input)
		if !matched {
			return fmt.Errorf("must contain only letters, numbers, _, -, .")
		}
		return nil
	}

	validateNotEmpty := func(input string) error {
		if input == "" {
			return fmt.Errorf("must not be empty")
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

	projectPrompt := promptui.Prompt{
		Label:    "Init: Project Name",
		Validate: validate,
	}
	project, err := projectPrompt.Run()
	if err != nil {
		return nil, err
	}

	authorPrompt := promptui.Prompt{
		Label:    "Init: Author Name",
		Validate: validateNotEmpty,
	}
	author, err := authorPrompt.Run()
	if err != nil {
		return nil, err
	}

	websitePrompt := promptui.Prompt{
		Label:    "Init: Author Website",
		Validate: validateURL,
	}
	website, err := websitePrompt.Run()
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

	zap.L().Info("packer.package.get.success", zap.String("project", project))
	return &PackageFile{
		Name: project,
		Author: PackageAuthor{
			Name:    author,
			Website: website,
		},
		Repository: repository,
	}, nil
}

func (pf *PackageFile) Find(url string) (*Package, error) {
	for _, p := range pf.Packages {
		if p.Source == url {
			zap.L().Debug("packer.package.find.hit", zap.String("name", p.Name))
			return p, nil
		}
	}
	zap.L().Warn("packer.package.find.miss", zap.String("url", url))
	return nil, fmt.Errorf("package not found")
}
