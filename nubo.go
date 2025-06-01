package nubo

import (
	"fmt"

	"github.com/nubolang/nubo/version"
)

type Nubo struct{}

func New() *Nubo {
	return &Nubo{}
}

func (n *Nubo) Version() string {
	return version.Version
}

func (n *Nubo) String() string {
	return fmt.Sprintf("Nubo %s", n.Version())
}

func (n *Nubo) Execute(file string) {
	// TODO
}
