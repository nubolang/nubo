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

func New(root string) (*Packer, error) {
	pkg, err := LoadPackageFile(root)
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
	baseDir, err := BaseDir()
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
	spin := pin.New(fmt.Sprintf("Installing %s", entry.Name),
		pin.WithSpinnerColor(pin.ColorCyan),
		pin.WithTextColor(pin.ColorYellow),
		pin.WithWriter(os.Stderr),
	)
	cancel := spin.Start(context.Background())
	defer cancel()
	defer spin.Stop(fmt.Sprintf("Done %s ✅", entry.Name))

	return entry.Download(baseDir)
}
