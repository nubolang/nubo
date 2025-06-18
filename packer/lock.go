package packer

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	git "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/nubolang/nubo/version"
	"gopkg.in/yaml.v3"
)

const LockYaml = "lock.yaml"

// LockEntry represents a Puff lock entry
type LockEntry struct {
	Name    string            `yaml:"name"` // user/repo
	Version string            `yaml:"version,omitempty"`
	Source  string            `yaml:"source,omitempty"` // example: https://github.com/user/repo
	Hash    string            `yaml:"hash,omitempty"`   // checksum, if any
	Meta    map[string]string `yaml:"meta,omitempty"`   // metadata
}

// LockFile (lock.yaml)
type LockFile struct {
	Version string       `yaml:"version"`
	Entries []*LockEntry `yaml:"entries"`
}

func LoadLockFile(root string) (*LockFile, error) {
	path := filepath.Join(root, LockYaml)

	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &LockFile{
				Version: version.Version,
			}, nil
		}
		return nil, err
	}
	defer file.Close()

	var lf LockFile
	if err := yaml.NewDecoder(file).Decode(&lf); err != nil {
		return nil, err
	}

	return &lf, nil
}

func (lf *LockFile) Save(root string) error {
	path := filepath.Join(root, LockYaml)
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	return yaml.NewEncoder(file).Encode(lf)
}

func (le *LockEntry) Download(baseCacheDir string) (string, error) {
	parts := strings.Split(le.Name, "/")
	if len(parts) < 2 {
		return "", fmt.Errorf("invalid package name: %s", le.Name)
	}
	user, repo := parts[0], parts[1]

	if le.Hash == "" {
		return "", fmt.Errorf("missing hash for package %s", le.Name)
	}

	dest := filepath.Join(baseCacheDir, le.Domain(), user, repo+"@"+le.Hash)
	if _, err := os.Stat(dest); err == nil {
		// Already exists
		return dest, nil
	}

	r, err := git.PlainClone(dest, false, &git.CloneOptions{
		URL:          le.Source,
		SingleBranch: true,
		Depth:        1,
	})
	if err != nil {
		return "", fmt.Errorf("failed to clone repo: %w", err)
	}

	w, err := r.Worktree()
	if err != nil {
		return "", err
	}

	hash := plumbing.NewHash(le.Hash)
	if err := w.Checkout(&git.CheckoutOptions{Hash: hash}); err != nil {
		return "", fmt.Errorf("failed to checkout hash: %w", err)
	}

	return dest, nil
}

func (le *LockEntry) Domain() string {
	uri, err := url.Parse(le.Source)
	if err != nil {
		return ""
	}
	return uri.Hostname()
}
