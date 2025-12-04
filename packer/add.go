package packer

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	git "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/yarlson/pin"
	"go.uber.org/zap"
)

func resolveHash(r *git.Repository, rev string) (plumbing.Hash, error) {
	h, err := r.ResolveRevision(plumbing.Revision(rev))
	if err != nil {
		return plumbing.ZeroHash, err
	}
	return *h, nil
}

func (p *Packer) Add(uri string) error {
	zap.L().Info("packer.add.start", zap.String("uri", uri))
	spin := pin.New(fmt.Sprintf("Adding %s", uri),
		pin.WithSpinnerColor(pin.ColorCyan),
		pin.WithTextColor(pin.ColorYellow),
		pin.WithWriter(os.Stderr),
	)
	cancel := spin.Start(context.Background())
	defer cancel()
	defer spin.Stop(fmt.Sprintf("Done %s", uri))

	if err := p.realAdd(uri); err != nil {
		zap.L().Error("packer.add.failed", zap.String("uri", uri), zap.Error(err))
		return err
	}

	zap.L().Info("packer.add.success", zap.String("uri", uri))
	return nil
}

func (p *Packer) realAdd(uri string) error {
	urlEntry, err := parseURI(uri)
	if err != nil {
		return err
	}

	domain := urlEntry.domain
	user := urlEntry.user
	repo := urlEntry.repo
	subpath := urlEntry.subpath
	version := urlEntry.version

	repoURL := fmt.Sprintf("https://%s/%s/%s.git", domain, user, repo)
	cachePath, err := PackageDir()
	if err != nil {
		zap.L().Error("packer.add.packageDir", zap.String("uri", uri), zap.Error(err))
		return err
	}

	zap.L().Debug("packer.add.repo", zap.String("uri", uri), zap.String("repoURL", repoURL), zap.String("version", version), zap.String("subpath", subpath))

	tmpDir := filepath.Join(cachePath, "__tmp__")
	cloneBasePath := filepath.Join(tmpDir, domain, user, repo)
	defer func() {
		os.RemoveAll(cloneBasePath)
		os.RemoveAll(tmpDir)
	}()

	// Open or clone repo
	var r *git.Repository
	if _, err := os.Stat(cloneBasePath); err == nil {
		r, err = git.PlainOpen(cloneBasePath)
		if err != nil {
			zap.L().Error("packer.add.openCache", zap.String("path", cloneBasePath), zap.Error(err))
			return err
		}
		err = r.Fetch(&git.FetchOptions{RemoteName: "origin"})
		if err != nil && err != git.NoErrAlreadyUpToDate {
			zap.L().Error("packer.add.fetch", zap.String("path", cloneBasePath), zap.Error(err))
			return err
		}
	} else if os.IsNotExist(err) {
		r, err = git.PlainClone(cloneBasePath, false, &git.CloneOptions{
			URL:          repoURL,
			SingleBranch: false,
			Depth:        0,
		})
		if err != nil {
			err = fmt.Errorf("git clone failed: %w", err)
			zap.L().Error("packer.add.clone", zap.String("url", repoURL), zap.Error(err))
			return err
		}
	} else {
		zap.L().Error("packer.add.statClonePath", zap.String("path", cloneBasePath), zap.Error(err))
		return err
	}

	w, err := r.Worktree()
	if err != nil {
		return err
	}

	if version == "latest" {
		headRef, err := r.Head()
		if err == nil && headRef != nil {
			version = headRef.Hash().String()
		} else {
			ref, err := r.Reference("refs/remotes/origin/HEAD", true)
			if err == nil {
				version = strings.TrimPrefix(ref.Target().String(), "refs/remotes/origin/")
			} else {
				version = "master" // final fallback
				zap.L().Warn("packer.add.latestFallback", zap.String("repo", repoURL))
			}
		}
	}

	hash, err := resolveHash(r, version)
	if err != nil {
		err = fmt.Errorf("cannot resolve revision %q: %w", version, err)
		zap.L().Error("packer.add.resolveRevision", zap.String("repo", repoURL), zap.String("version", version), zap.Error(err))
		return err
	}

	shortHash := hash.String()[:7]
	finalPath := filepath.Join(cachePath, domain, user, repo+"@"+hash.String())
	zap.L().Debug("packer.add.resolved", zap.String("repo", repoURL), zap.String("hash", hash.String()), zap.String("finalPath", finalPath))

	if _, err := os.Stat(finalPath); err == nil {
		// cached version exists, done
		if err := os.RemoveAll(cloneBasePath); err != nil {
			zap.L().Warn("packer.add.cleanup", zap.String("path", cloneBasePath), zap.Error(err))
			return err
		}

		zap.L().Info("packer.add.cached", zap.String("repo", repoURL), zap.String("hash", hash.String()))
		return p.updatePackageFiles(user, repo, subpath, repoURL, hash.String(), shortHash, finalPath)
	}

	err = w.Checkout(&git.CheckoutOptions{Hash: hash})
	if err != nil {
		err = fmt.Errorf("checkout failed: %w", err)
		zap.L().Error("packer.add.checkout", zap.String("repo", repoURL), zap.String("hash", hash.String()), zap.Error(err))
		return err
	}

	if err := os.MkdirAll(filepath.Dir(finalPath), 0755); err != nil {
		zap.L().Error("packer.add.mkdir", zap.String("path", filepath.Dir(finalPath)), zap.Error(err))
		return err
	}

	if err := os.Rename(cloneBasePath, finalPath); err != nil {
		zap.L().Error("packer.add.rename", zap.String("from", cloneBasePath), zap.String("to", finalPath), zap.Error(err))
		return err
	}

	// Update Package with short hash and LockEntry with long hash
	if err := p.updatePackageFiles(user, repo, subpath, repoURL, hash.String(), shortHash, finalPath); err != nil {
		return err
	}

	finalPathPkg := finalPath
	if subpath != "" {
		finalPathPkg = filepath.Join(finalPathPkg, subpath)
	}

	finalPathPkg = filepath.Join(finalPathPkg, LockYaml)
	entries, err := p.Load(finalPathPkg, cachePath)
	if err != nil {
		zap.L().Error("packer.add.loadNested", zap.String("path", finalPathPkg), zap.Error(err))
		return err
	}

	p.Lock.Entries = append(p.Lock.Entries, entries...)
	if err := p.Lock.Save(p.root); err != nil {
		zap.L().Error("packer.add.saveLock", zap.Error(err))
		return err
	}

	zap.L().Info("packer.add.completed", zap.String("repo", repoURL), zap.String("hash", hash.String()))
	return nil
}

func (p *Packer) updatePackageFiles(user, repo, subpath, repoURL, hash, shortHash, finalPath string) error {
	pkgName := user + "/" + repo
	if subpath != "" {
		pkgName += "/" + subpath
	}

	found := false
	for _, pkg := range p.Package.Packages {
		if pkg.Name == pkgName {
			pkg.CommitHashShort = shortHash
			found = true
			break
		}
	}
	if !found {
		zap.L().Debug("packer.add.newPackageEntry", zap.String("name", pkgName))
		p.Package.Packages = append(p.Package.Packages, &Package{
			Name:            pkgName,
			CommitHashShort: shortHash,
			Source:          repoURL,
		})
	}
	if err := p.Package.Save(p.root); err != nil {
		zap.L().Error("packer.add.savePackage", zap.Error(err))
		return err
	}

	folderHash, err := hashDir(finalPath)
	if err != nil {
		zap.L().Error("packer.add.hashDir", zap.String("path", finalPath), zap.Error(err))
		return err
	}
	zap.L().Debug("packer.add.folderHash", zap.String("name", pkgName), zap.String("hash", folderHash))

	foundLock := false
	for _, entry := range p.Lock.Entries {
		if entry.Name == pkgName {
			entry.Source = repoURL
			entry.CommitHash = hash
			entry.Hash = "sha256:" + folderHash
			foundLock = true
			zap.L().Debug("packer.add.updateLockEntry", zap.String("name", pkgName))
			break
		}
	}
	if !foundLock {
		zap.L().Debug("packer.add.newLockEntry", zap.String("name", pkgName))
		p.Lock.Entries = append(p.Lock.Entries, &LockEntry{
			Name:       pkgName,
			Source:     repoURL,
			CommitHash: hash,
			Hash:       "sha256:" + folderHash,
		})
	}

	if err := p.Lock.Save(p.root); err != nil {
		zap.L().Error("packer.add.saveLockEntry", zap.Error(err))
		return err
	}

	zap.L().Debug("packer.add.metadataSaved", zap.String("name", pkgName))
	return nil
}

type parsedUrlEntry struct {
	domain  string
	user    string
	repo    string
	version string
	subpath string
}

func parseURI(uri string) (*parsedUrlEntry, error) {
	uri = strings.TrimPrefix(uri, "https://")
	uri = strings.TrimPrefix(uri, "http://")
	atIdx := strings.LastIndex(uri, "@")

	var source, version string
	if atIdx == -1 {
		source = uri
		version = "latest"
	} else {
		source = uri[:atIdx]
		version = uri[atIdx+1:]
		if version == "" {
			version = "latest"
		}
	}

	parts := strings.Split(source, "/")
	if len(parts) < 3 {
		return nil, fmt.Errorf("invalid source format")
	}
	domain, user, repo := parts[0], parts[1], parts[2]
	subpath := ""
	if len(parts) > 3 {
		subpath = strings.Join(parts[3:], "/")
	}

	return &parsedUrlEntry{
		domain:  domain,
		user:    user,
		repo:    repo,
		version: version,
		subpath: subpath,
	}, nil
}
