package packer

import (
	"context"
	"fmt"
	"os"

	"github.com/fatih/color"
	"github.com/yarlson/pin"
	"go.uber.org/zap"
)

// Packer is Nubo's package manager
type Packer struct {
	root string

	Package *PackageFile
	Lock    *LockFile
}

func Init(root string) (*Packer, error) {
	pkg, err := LoadPackageFile(root, true)
	if err != nil {
		return nil, err
	}

	lock, err := LoadLockFile(root)
	if err != nil {
		return nil, err
	}

	return &Packer{
		root:    root,
		Package: pkg,
		Lock:    lock,
	}, nil
}

func New(root string, force ...bool) (*Packer, error) {
	var f bool
	if len(force) > 0 {
		f = force[0]
	}

	pkg, err := LoadPackageFile(root, f)
	if err != nil {
		return nil, err
	}

	lock, err := LoadLockFile(root)
	if err != nil {
		return nil, err
	}

	return &Packer{
		root:    root,
		Package: pkg,
		Lock:    lock,
	}, nil
}

func (p *Packer) Download() error {
	zap.L().Info("packer.download.start", zap.Int("entries", len(p.Lock.Entries)))
	baseDir, err := PackageDir()
	if err != nil {
		zap.L().Error("packer.download.packageDir", zap.Error(err))
		return err
	}

	for _, entry := range p.Lock.Entries {
		if _, err := p.downloadEntry(entry, baseDir); err != nil {
			zap.L().Error("packer.download.entryFailed", zap.String("name", entry.Name), zap.Error(err))
			return err
		}
	}

	color.Green("Downloaded %d packages", len(p.Lock.Entries))
	zap.L().Info("packer.download.success", zap.Int("entries", len(p.Lock.Entries)))
	return nil
}

func (p *Packer) downloadEntry(entry *LockEntry, baseDir string) (string, error) {
	zap.L().Debug("packer.download.entry", zap.String("name", entry.Name), zap.String("hash", entry.CommitHash))
	spin := pin.New(fmt.Sprintf("Installing %s\n", entry.Name),
		pin.WithSpinnerColor(pin.ColorCyan),
		pin.WithTextColor(pin.ColorYellow),
		pin.WithWriter(os.Stderr),
	)
	cancel := spin.Start(context.Background())
	defer cancel()

	dir, local, err := entry.Download(baseDir)
	if err != nil {
		zap.L().Error("packer.download.cloneFailed", zap.String("name", entry.Name), zap.Error(err))
		return "", err
	}

	spin.UpdateMessage("Validating package 🕷️")
	hash, err := hashDir(dir)
	if err != nil {
		return "", err
	}

	if entry.Hash != "sha256:"+hash {
		spin.Fail(fmt.Sprintf("Failed to validate %s 🐛", entry.Name))
		fmt.Println(entry.Hash)
		fmt.Println("sha256:" + hash)
		fmt.Println(dir)
		zap.L().Error("packer.download.hashMismatch", zap.String("name", entry.Name), zap.String("expected", entry.Hash), zap.String("actual", "sha256:"+hash))
		return "", fmt.Errorf("invalid hash for %s", entry.Name)
	}

	var emoji string = " ✅"
	if local {
		emoji = color.New(color.FgHiCyan).Sprint(" (cached) 📦")
	}

	spin.Stop(fmt.Sprintf("Done %s%s", entry.Name, emoji))
	zap.L().Debug("packer.download.entryDone", zap.String("name", entry.Name))
	return dir, nil
}

func (p *Packer) Write() error {
	if err := p.Package.Save(p.root); err != nil {
		zap.L().Error("packer.write.package", zap.Error(err))
		return err
	}

	if err := p.Lock.Save(p.root); err != nil {
		zap.L().Error("packer.write.lock", zap.Error(err))
		return err
	}

	zap.L().Debug("packer.write.success")

	return nil
}
