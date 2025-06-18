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
)

func resolveHash(r *git.Repository, rev string) (plumbing.Hash, error) {
	h, err := r.ResolveRevision(plumbing.Revision(rev))
	if err != nil {
		return plumbing.ZeroHash, err
	}
	return *h, nil
}

func (p *Packer) Add(uri string) error {
	spin := pin.New(fmt.Sprintf("Adding %s", uri),
		pin.WithSpinnerColor(pin.ColorCyan),
		pin.WithTextColor(pin.ColorYellow),
		pin.WithWriter(os.Stderr),
	)
	cancel := spin.Start(context.Background())
	defer cancel()
	defer spin.Stop(fmt.Sprintf("Done %s", uri))

	return p.realAdd(uri)
}

func (p *Packer) realAdd(uri string) error {
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
		return fmt.Errorf("invalid source format")
	}
	domain, user, repo := parts[0], parts[1], parts[2]
	subpath := ""
	if len(parts) > 3 {
		subpath = strings.Join(parts[3:], "/")
	}

	repoURL := fmt.Sprintf("https://%s/%s/%s.git", domain, user, repo)
	cachePath, err := PackageDir()
	if err != nil {
		return err
	}

	cloneBasePath := filepath.Join(cachePath, "__tmp__", domain, user, repo)

	// Open or clone repo
	var r *git.Repository
	if _, err := os.Stat(cloneBasePath); err == nil {
		r, err = git.PlainOpen(cloneBasePath)
		if err != nil {
			return err
		}
		err = r.Fetch(&git.FetchOptions{RemoteName: "origin"})
		if err != nil && err != git.NoErrAlreadyUpToDate {
			return err
		}
	} else if os.IsNotExist(err) {
		r, err = git.PlainClone(cloneBasePath, false, &git.CloneOptions{
			URL:          repoURL,
			SingleBranch: false,
			Depth:        0,
		})
		if err != nil {
			return fmt.Errorf("git clone failed: %w", err)
		}
	} else {
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
			version = "master"
		}
	}

	hash, err := resolveHash(r, version)
	if err != nil {
		return fmt.Errorf("cannot resolve revision %q: %w", version, err)
	}

	shortHash := hash.String()[:7]
	finalPath := filepath.Join(cachePath, domain, user, repo+"@"+hash.String())

	if _, err := os.Stat(finalPath); err == nil {
		// cached version exists, done
		if err := os.RemoveAll(cloneBasePath); err != nil {
			return err
		}

		return p.updatePackageFiles(domain, user, repo, subpath, repoURL, hash.String(), shortHash)
	}

	err = w.Checkout(&git.CheckoutOptions{Hash: hash})
	if err != nil {
		return fmt.Errorf("checkout failed: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(finalPath), 0755); err != nil {
		return err
	}

	if err := os.Rename(cloneBasePath, finalPath); err != nil {
		return err
	}

	// Update Package with short hash and LockEntry with long hash
	if err := p.updatePackageFiles(domain, user, repo, subpath, repoURL, hash.String(), shortHash); err != nil {
		return err
	}

	finalPathPkg := finalPath
	if subpath != "" {
		finalPathPkg = filepath.Join(finalPathPkg, subpath)
	}

	finalPathPkg = filepath.Join(finalPathPkg, LockYaml)
	entries, err := p.Load(finalPathPkg, cachePath)
	if err != nil {
		return err
	}

	p.Lock.Entries = append(p.Lock.Entries, entries...)
	if err := p.Lock.Save(p.root); err != nil {
		return err
	}

	return nil
}

func (p *Packer) updatePackageFiles(domain, user, repo, subpath, repoURL, hash, shortHash string) error {
	pkgName := user + "/" + repo
	if subpath != "" {
		pkgName += "/" + subpath
	}

	found := false
	for _, pkg := range p.Package.Packages {
		if pkg.Name == pkgName {
			pkg.Version = shortHash
			found = true
			break
		}
	}
	if !found {
		p.Package.Packages = append(p.Package.Packages, &Package{
			Name:    pkgName,
			Version: shortHash,
		})
	}
	if err := p.Package.Save(p.root); err != nil {
		return err
	}

	foundLock := false
	for _, entry := range p.Lock.Entries {
		if entry.Name == pkgName {
			entry.Version = shortHash
			entry.Source = repoURL
			entry.Hash = hash
			foundLock = true
			break
		}
	}
	if !foundLock {
		p.Lock.Entries = append(p.Lock.Entries, &LockEntry{
			Name:    pkgName,
			Version: shortHash,
			Source:  repoURL,
			Hash:    hash,
		})
	}

	return p.Lock.Save(p.root)
}
