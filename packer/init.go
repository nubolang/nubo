package packer

func (p *Packer) Init(name string) error {
	p.Package.Name = name

	if err := p.Package.Save(p.root); err != nil {
		return err
	}

	return p.Lock.Save(p.root)
}
