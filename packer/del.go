package packer

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/yarlson/pin"
)

func (p *Packer) Del(uri string, cleanUp bool) error {
	spin := pin.New(fmt.Sprintf("Deleting %s", uri),
		pin.WithSpinnerColor(pin.ColorCyan),
		pin.WithTextColor(pin.ColorYellow),
		pin.WithWriter(os.Stderr),
	)
	cancel := spin.Start(context.Background())
	defer cancel()
	defer spin.Stop(fmt.Sprintf("Done %s", uri))

	if err := p.realDel(uri, cleanUp); err != nil {
		return err
	}

	return p.Write()
}

func (p *Packer) realDel(uri string, cleanUp bool) error {
	urlEntry, err := parseURI(uri)
	if err != nil {
		return err
	}

	repoURL := fmt.Sprintf("https://%s/%s/%s.git", urlEntry.domain, urlEntry.user, urlEntry.repo)
	pkg, err := p.Package.Find(repoURL)
	if err != nil {
		return err
	}

	lock, err := p.Lock.Find(repoURL)
	if err != nil {
		return err
	}

	// Count references from all other packages
	var total int
	for _, entry := range p.Lock.Entries {
		if entry.Source == lock.Source && entry.CommitHash == lock.CommitHash {
			continue
		}
		c, err := p.countMap(entry, lock, make(map[string]bool))
		if err != nil {
			return err
		}
		total += c
	}

	cachePath, err := PackageDir()
	if err != nil {
		return err
	}

	rawSource := filepath.Join(cachePath, urlEntry.domain, urlEntry.user, urlEntry.repo+"@"+lock.CommitHash)

	// If referenced, only remove from package metadata
	if total > 0 {
		p.removeFromPackage(pkg)
		return p.Package.Save(p.root)
	}

	// Check all dependencies before deleting
	sourcePathYaml := filepath.Join(rawSource, LockYaml)
	if _, err := os.Stat(sourcePathYaml); err == nil {
		entries, err := p.Load(sourcePathYaml, cachePath)
		if err != nil {
			return err
		}

		safeToDelete := true
		for _, entry := range p.Lock.Entries {
			has, err := p.hasDepMap(entry, entries, make(map[string]bool))
			if err != nil {
				return err
			}
			if has {
				safeToDelete = false
				break
			}
		}

		if !safeToDelete {
			p.removeFromPackage(pkg)
			return p.Package.Save(p.root)
		}

		// Recursively delete nested dependencies
		for _, dep := range entries {
			if err := p.realDel(dep.Source, cleanUp); err != nil {
				return err
			}
		}
	}

	// Remove package from disk
	if cleanUp {
		if err := os.RemoveAll(rawSource); err != nil {
			return err
		}
	}

	// Remove from metadata
	p.removeFromPackage(pkg)
	p.removeFromLock(lock)
	return nil
}

func (p *Packer) countMap(lock *LockEntry, delEntry *LockEntry, visited map[string]bool) (int, error) {
	cachePath, err := PackageDir()
	if err != nil {
		return 0, err
	}

	key := lock.Source + "@" + lock.CommitHash
	if visited[key] {
		return 0, nil
	}
	visited[key] = true

	urlEntry, err := parseURI(lock.Source)
	if err != nil {
		return 0, err
	}

	sourcePath := filepath.Join(cachePath, urlEntry.domain, urlEntry.user, urlEntry.repo+"@"+lock.CommitHash)
	if urlEntry.repo != lock.Name {
		sourcePath = filepath.Join(sourcePath, strings.TrimPrefix(lock.Name, urlEntry.repo+"/"))
	}
	sourcePath = filepath.Join(sourcePath, LockYaml)

	if _, err := os.Stat(sourcePath); err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, err
	}

	entries, err := p.Load(sourcePath, cachePath)
	if err != nil {
		return 0, err
	}

	var count int
	for _, entry := range entries {
		if entry.Source == delEntry.Source && entry.CommitHash == delEntry.CommitHash {
			count++
			continue
		}

		c, err := p.countMap(entry, delEntry, visited)
		if err != nil {
			return 0, err
		}
		count += c
	}

	return count, nil
}

func (p *Packer) hasDepMap(lock *LockEntry, dependencies []*LockEntry, visited map[string]bool) (bool, error) {
	cachePath, err := PackageDir()
	if err != nil {
		return false, err
	}

	key := lock.Source + "@" + lock.CommitHash
	if visited[key] {
		return false, nil
	}
	visited[key] = true

	urlEntry, err := parseURI(lock.Source)
	if err != nil {
		return false, err
	}

	sourcePath := filepath.Join(cachePath, urlEntry.domain, urlEntry.user, urlEntry.repo+"@"+lock.CommitHash)
	if urlEntry.repo != lock.Name {
		sourcePath = filepath.Join(sourcePath, strings.TrimPrefix(lock.Name, urlEntry.repo+"/"))
	}
	sourcePath = filepath.Join(sourcePath, LockYaml)

	if _, err := os.Stat(sourcePath); err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}

	entries, err := p.Load(sourcePath, cachePath)
	if err != nil {
		return false, err
	}

	for _, entry := range entries {
		for _, dep := range dependencies {
			if entry.Source == dep.Source && entry.CommitHash == dep.CommitHash {
				return true, nil
			}

			has, err := p.hasDepMap(entry, dependencies, visited)
			if err != nil {
				return false, err
			}
			if has {
				return true, nil
			}
		}
	}

	return false, nil
}

func (p *Packer) removeFromPackage(pkg *Package) {
	var newPkgs []*Package
	for _, pck := range p.Package.Packages {
		if pck.Source == pkg.Source && pck.CommitHashShort == pkg.CommitHashShort {
			continue
		}
		newPkgs = append(newPkgs, pck)
	}
	p.Package.Packages = newPkgs
}

func (p *Packer) removeFromLock(lock *LockEntry) {
	var newEntries []*LockEntry
	for _, entry := range p.Lock.Entries {
		if entry.Source == lock.Source && entry.CommitHash == lock.CommitHash {
			continue
		}
		newEntries = append(newEntries, entry)
	}
	p.Lock.Entries = newEntries
}
