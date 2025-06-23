package n

import (
	"fmt"
	"sync"

	"github.com/nubolang/nubo/internal/debug"
	"github.com/nubolang/nubo/language"
)

type Package struct {
	Name  string
	proto language.Prototype
	debug *debug.Debug
}

func NewPackage(name string, dg *debug.Debug) *Package {
	return &Package{
		Name:  name,
		debug: dg,
	}
}

func (i *Package) ID() string {
	return fmt.Sprintf("%p", i)
}

func (i *Package) Type() *language.Type {
	return TStruct
}

func (i *Package) Inspect() string {
	return fmt.Sprintf("<Object(@std/%s)>", i.Name)
}

func (i *Package) TypeString() string {
	return fmt.Sprintf("<Object(@std/%s)>", i.Name)
}

func (i *Package) String() string {
	return fmt.Sprintf("@std/%s", i.Name)
}

func (i *Package) GetPrototype() language.Prototype {
	if i.proto == nil {
		i.proto = NewPackagePrototype(i)
	}
	return i.proto
}

func (i *Package) Value() any {
	return nil
}

func (i *Package) Debug() *debug.Debug {
	return nil
}

func (i *Package) Clone() language.Object {
	return i
}

type PackagePrototype struct {
	base *Package
	data map[string]language.Object
	mu   sync.RWMutex
}

func NewPackagePrototype(base *Package) *PackagePrototype {
	ip := &PackagePrototype{
		base: base,
		data: make(map[string]language.Object),
	}

	return ip
}

func (i *PackagePrototype) GetObject(name string) (language.Object, bool) {
	i.mu.RLock()
	defer i.mu.RUnlock()
	obj, ok := i.data[name]
	return obj, ok
}

func (i *PackagePrototype) SetObject(name string, value language.Object) error {
	i.mu.Lock()
	defer i.mu.Unlock()
	i.data[name] = value
	return nil
}

func (i *PackagePrototype) Objects() map[string]language.Object {
	i.mu.RLock()
	defer i.mu.RUnlock()
	return i.data
}
