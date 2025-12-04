package packer

import (
	"context"
	"fmt"
	"os"

	"github.com/fatih/color"
	"github.com/yarlson/pin"
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
	baseDir, err := PackageDir()
	if err != nil {
		return err
	}

	for _, entry := range p.Lock.Entries {
		if _, err := p.downloadEntry(entry, baseDir); err != nil {
			return err
		}
	}

	color.Green("Downloaded %d packages", len(p.Lock.Entries))
	return nil
}

func (p *Packer) downloadEntry(entry *LockEntry, baseDir string) (string, error) {
	spin := pin.New(fmt.Sprintf("Installing %s\n", entry.Name),
		pin.WithSpinnerColor(pin.ColorCyan),
		pin.WithTextColor(pin.ColorYellow),
		pin.WithWriter(os.Stderr),
	)
	cancel := spin.Start(context.Background())
	defer cancel()

	dir, err := entry.Download(baseDir)
	if err != nil {
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
		return "", fmt.Errorf("invalid hash for %s", entry.Name)
	}

	spin.Stop(fmt.Sprintf("Done %s ✅", entry.Name))
	return dir, nil
}

func (p *Packer) Write() error {
	if err := p.Package.Save(p.root); err != nil {
		return err
	}

	if err := p.Lock.Save(p.root); err != nil {
		return err
	}

	return nil
}
