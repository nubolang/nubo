package packer

func (p *Packer) Init(author string) error {
	p.Package.Author = author

	if err := p.Package.Save(p.root); err != nil {
		return err
	}

	return p.Lock.Save(p.root)
}
