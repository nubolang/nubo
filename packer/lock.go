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
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

const LockYaml = "lock.yaml"

// LockEntry represents a Puff lock entry
type LockEntry struct {
	Name       string            `yaml:"name"`                  // user/repo
	Source     string            `yaml:"source,omitempty"`      // example: https://github.com/user/repo
	CommitHash string            `yaml:"commit_hash,omitempty"` // commit hash
	Hash       string            `yaml:"hash,omitempty"`        // checksum, if any
	Meta       map[string]string `yaml:"meta,omitempty"`        // metadata
}

const LockVersion = "1"

// LockFile (lock.yaml)
type LockFile struct {
	Version     string       `yaml:"version"`
	NuboVersion string       `yaml:"nubo_version"`
	Entries     []*LockEntry `yaml:"entries"`
}

func LoadLockFile(root string) (*LockFile, error) {
	path := filepath.Join(root, LockYaml)

	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			zap.L().Info("packer.lock.load.new", zap.String("path", path))
			return &LockFile{
				Version:     LockVersion,
				NuboVersion: version.Version,
			}, nil
		}
		zap.L().Error("packer.lock.load.open", zap.String("path", path), zap.Error(err))
		return nil, err
	}
	defer file.Close()

	var lf LockFile
	if err := yaml.NewDecoder(file).Decode(&lf); err != nil {
		zap.L().Error("packer.lock.load.decode", zap.String("path", path), zap.Error(err))
		return nil, err
	}

	zap.L().Debug("packer.lock.load.success", zap.Int("entries", len(lf.Entries)))
	return &lf, nil
}

func (lf *LockFile) Save(root string) error {
	path := filepath.Join(root, LockYaml)
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		zap.L().Error("packer.lock.save.open", zap.String("path", path), zap.Error(err))
		return err
	}
	defer file.Close()

	if err := yaml.NewEncoder(file).Encode(lf); err != nil {
		zap.L().Error("packer.lock.save.encode", zap.String("path", path), zap.Error(err))
		return err
	}

	zap.L().Debug("packer.lock.save.success", zap.String("path", path), zap.Int("entries", len(lf.Entries)))
	return nil
}

func (le *LockEntry) Download(baseCacheDir string) (string, error) {
	parts := strings.Split(le.Name, "/")
	if len(parts) < 2 {
		err := fmt.Errorf("invalid package name: %s", le.Name)
		zap.L().Error("packer.lock.download.invalidName", zap.String("name", le.Name))
		return "", err
	}
	user, repo := parts[0], parts[1]

	if le.CommitHash == "" {
		err := fmt.Errorf("missing hash for package %s", le.Name)
		zap.L().Error("packer.lock.download.noHash", zap.String("name", le.Name))
		return "", err
	}

	dest := filepath.Join(baseCacheDir, le.Domain(), user, repo+"@"+le.CommitHash)
	if _, err := os.Stat(dest); err == nil {
		// Already exists
		zap.L().Debug("packer.lock.download.cached", zap.String("name", le.Name), zap.String("dest", dest))
		return dest, nil
	}

	r, err := git.PlainClone(dest, false, &git.CloneOptions{
		URL:          le.Source,
		SingleBranch: true,
		Depth:        1,
	})
	if err != nil {
		err = fmt.Errorf("failed to clone repo: %w", err)
		zap.L().Error("packer.lock.download.clone", zap.String("source", le.Source), zap.Error(err))
		return "", err
	}

	w, err := r.Worktree()
	if err != nil {
		zap.L().Error("packer.lock.download.worktree", zap.String("source", le.Source), zap.Error(err))
		return "", err
	}

	hash := plumbing.NewHash(le.CommitHash)
	if err := w.Checkout(&git.CheckoutOptions{Hash: hash}); err != nil {
		err = fmt.Errorf("failed to checkout hash: %w", err)
		zap.L().Error("packer.lock.download.checkout", zap.String("source", le.Source), zap.Error(err))
		return "", err
	}

	zap.L().Debug("packer.lock.download.success", zap.String("name", le.Name), zap.String("dest", dest))
	return dest, nil
}

func (le *LockEntry) Domain() string {
	uri, err := url.Parse(le.Source)
	if err != nil {
		return ""
	}
	return uri.Hostname()
}

func (lf *LockFile) Find(url string) (*LockEntry, error) {
	for _, entry := range lf.Entries {
		if entry.Source == url {
			zap.L().Debug("packer.lock.find.hit", zap.String("name", entry.Name))
			return entry, nil
		}
	}
	err := fmt.Errorf("package not found")
	zap.L().Warn("packer.lock.find.miss", zap.String("url", url))
	return nil, err
}
