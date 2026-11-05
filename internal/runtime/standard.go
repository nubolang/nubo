package runtime

import (
	"context"

	"github.com/nubolang/nubo/internal/debug"
	"github.com/nubolang/nubo/internal/gobridge"
	"github.com/nubolang/nubo/internal/packages"
	"github.com/nubolang/nubo/language"
	"github.com/nubolang/nubo/native/n"
)

func (r *Runtime) importSourceBuiltin(ctx context.Context, name string, definition packages.SourceDefinition) (language.Object, error) {
	native, err := gobridge.WrapPackage(name, definition.Funcs)
	if err != nil {
		return nil, err
	}

	mod, err := newModule(name, native, definition.Source, definition.Files)
	if err != nil {
		return nil, err
	}

	r.mu.Lock()
	if existing := r.modules[name]; existing != nil {
		mod = existing
	} else {
		r.modules[name] = mod
	}
	r.mu.Unlock()

	return r.loadModule(ctx, mod)
}

func (r *Runtime) importStandard(ctx context.Context, dg *debug.Debug) (language.Object, error) {
	pkg := n.NewPackage("@std", dg)
	proto := pkg.GetPrototype()
	for _, name := range packages.Names() {
		object, ok, err := r.ImportPackage(ctx, packages.BuiltInModulePrefix+name, dg)
		if err != nil {
			return nil, err
		}
		if !ok {
			continue
		}

		if err := proto.SetObject(ctx, name, object); err != nil {
			return nil, err
		}
	}

	return pkg, nil
}
