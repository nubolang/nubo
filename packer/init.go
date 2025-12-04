package packer

import "go.uber.org/zap"

func (p *Packer) Init() error {
	zap.L().Info("packer.init.start")
	if err := p.Package.Save(p.root); err != nil {
		zap.L().Error("packer.init.package", zap.Error(err))
		return err
	}

	if err := p.Lock.Save(p.root); err != nil {
		zap.L().Error("packer.init.lock", zap.Error(err))
		return err
	}

	zap.L().Info("packer.init.success")
	return nil
}
