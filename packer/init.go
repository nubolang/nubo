package packer

func (p *Packer) Init() error {
	if err := p.Package.Save(p.root); err != nil {
		return err
	}

	return p.Lock.Save(p.root)
}
