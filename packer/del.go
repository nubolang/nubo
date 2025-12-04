package packer

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/yarlson/pin"
	"go.uber.org/zap"
)

func (p *Packer) Del(uri string, cleanUp bool) error {
	zap.L().Info("packer.del.start", zap.String("uri", uri), zap.Bool("cleanup", cleanUp))
	spin := pin.New(fmt.Sprintf("Deleting %s", uri),
		pin.WithSpinnerColor(pin.ColorCyan),
		pin.WithTextColor(pin.ColorYellow),
		pin.WithWriter(os.Stderr),
	)
	cancel := spin.Start(context.Background())
	defer cancel()
	defer spin.Stop(fmt.Sprintf("Done %s", uri))

	if err := p.realDel(uri, cleanUp); err != nil {
		zap.L().Error("packer.del.failed", zap.String("uri", uri), zap.Error(err))
		return err
	}

	if err := p.Write(); err != nil {
		zap.L().Error("packer.del.writeFailed", zap.String("uri", uri), zap.Error(err))
		return err
	}

	zap.L().Info("packer.del.success", zap.String("uri", uri))
	return nil
}

func (p *Packer) realDel(uri string, cleanUp bool) error {
	urlEntry, err := parseURI(uri)
	if err != nil {
		zap.L().Error("packer.del.parse", zap.String("uri", uri), zap.Error(err))
		return err
	}

	repoURL := fmt.Sprintf("https://%s/%s/%s.git", urlEntry.domain, urlEntry.user, urlEntry.repo)
	pkg, err := p.Package.Find(repoURL)
	if err != nil {
		zap.L().Error("packer.del.pkgNotFound", zap.String("repo", repoURL), zap.Error(err))
		return err
	}

	lock, err := p.Lock.Find(repoURL)
	if err != nil {
		zap.L().Error("packer.del.lockNotFound", zap.String("repo", repoURL), zap.Error(err))
		return err
	}

	zap.L().Debug("packer.del.target", zap.String("repo", repoURL), zap.String("commit", lock.CommitHash))

	// Count references from all other packages
	var total int
	for _, entry := range p.Lock.Entries {
		if entry.Source == lock.Source && entry.CommitHash == lock.CommitHash {
			continue
		}
		c, err := p.countMap(entry, lock, make(map[string]bool))
		if err != nil {
			zap.L().Error("packer.del.countRefFailed", zap.String("entry", entry.Name), zap.Error(err))
			return err
		}
		total += c
	}
	zap.L().Debug("packer.del.references", zap.String("repo", repoURL), zap.Int("count", total))

	cachePath, err := PackageDir()
	if err != nil {
		zap.L().Error("packer.del.packageDir", zap.Error(err))
		return err
	}

	rawSource := filepath.Join(cachePath, urlEntry.domain, urlEntry.user, urlEntry.repo+"@"+lock.CommitHash)

	// If referenced, only remove from package metadata
	if total > 0 {
		zap.L().Info("packer.del.referenced", zap.String("repo", repoURL), zap.Int("references", total))
		p.removeFromPackage(pkg)
		return p.Package.Save(p.root)
	}

	// Check all dependencies before deleting
	sourcePathYaml := filepath.Join(rawSource, LockYaml)
	if _, err := os.Stat(sourcePathYaml); err == nil {
		entries, err := p.Load(sourcePathYaml, cachePath)
		if err != nil {
			zap.L().Error("packer.del.loadDeps", zap.String("path", sourcePathYaml), zap.Error(err))
			return err
		}

		safeToDelete := true
		for _, entry := range p.Lock.Entries {
			has, err := p.hasDepMap(entry, entries, make(map[string]bool))
			if err != nil {
				zap.L().Error("packer.del.hasDepMap", zap.String("entry", entry.Name), zap.Error(err))
				return err
			}
			if has {
				safeToDelete = false
				break
			}
		}

		if !safeToDelete {
			zap.L().Info("packer.del.dependencyInUse", zap.String("repo", repoURL))
			p.removeFromPackage(pkg)
			return p.Package.Save(p.root)
		}

		// Recursively delete nested dependencies
		for _, dep := range entries {
			if err := p.realDel(dep.Source, cleanUp); err != nil {
				zap.L().Error("packer.del.recursive", zap.String("dep", dep.Name), zap.Error(err))
				return err
			}
		}
	}

	// Remove package from disk
	if cleanUp {
		if err := os.RemoveAll(rawSource); err != nil {
			zap.L().Error("packer.del.cleanup", zap.String("path", rawSource), zap.Error(err))
			return err
		}
		zap.L().Debug("packer.del.removedCache", zap.String("path", rawSource))
	}

	// Remove from metadata
	p.removeFromPackage(pkg)
	p.removeFromLock(lock)
	zap.L().Info("packer.del.removed", zap.String("repo", repoURL))
	return nil
}

func (p *Packer) countMap(lock *LockEntry, delEntry *LockEntry, visited map[string]bool) (int, error) {
	cachePath, err := PackageDir()
	if err != nil {
		zap.L().Error("packer.del.countMap.packageDir", zap.Error(err))
		return 0, err
	}

	key := lock.Source + "@" + lock.CommitHash
	if visited[key] {
		return 0, nil
	}
	visited[key] = true
	zap.L().Debug("packer.del.countMap.visit", zap.String("key", key))

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
		zap.L().Error("packer.del.countMap.stat", zap.String("path", sourcePath), zap.Error(err))
		return 0, err
	}

	entries, err := p.Load(sourcePath, cachePath)
	if err != nil {
		zap.L().Error("packer.del.countMap.load", zap.String("path", sourcePath), zap.Error(err))
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
			zap.L().Error("packer.del.countMap.recursive", zap.String("entry", entry.Name), zap.Error(err))
			return 0, err
		}
		count += c
	}

	return count, nil
}

func (p *Packer) hasDepMap(lock *LockEntry, dependencies []*LockEntry, visited map[string]bool) (bool, error) {
	cachePath, err := PackageDir()
	if err != nil {
		zap.L().Error("packer.del.hasDepMap.packageDir", zap.Error(err))
		return false, err
	}

	key := lock.Source + "@" + lock.CommitHash
	if visited[key] {
		return false, nil
	}
	visited[key] = true
	zap.L().Debug("packer.del.hasDepMap.visit", zap.String("key", key))

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
		zap.L().Error("packer.del.hasDepMap.stat", zap.String("path", sourcePath), zap.Error(err))
		return false, err
	}

	entries, err := p.Load(sourcePath, cachePath)
	if err != nil {
		zap.L().Error("packer.del.hasDepMap.load", zap.String("path", sourcePath), zap.Error(err))
		return false, err
	}

	for _, entry := range entries {
		for _, dep := range dependencies {
			if entry.Source == dep.Source && entry.CommitHash == dep.CommitHash {
				return true, nil
			}

			has, err := p.hasDepMap(entry, dependencies, visited)
			if err != nil {
				zap.L().Error("packer.del.hasDepMap.recursive", zap.String("entry", entry.Name), zap.Error(err))
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
			zap.L().Debug("packer.del.removePackageEntry", zap.String("name", pck.Name))
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
			zap.L().Debug("packer.del.removeLockEntry", zap.String("name", entry.Name))
			continue
		}
		newEntries = append(newEntries, entry)
	}
	p.Lock.Entries = newEntries
}
