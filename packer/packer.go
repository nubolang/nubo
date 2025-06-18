package packer

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
