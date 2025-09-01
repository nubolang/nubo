package system

import (
	"os"

	"github.com/nubolang/nubo/native/n"
)

var cwd = n.Function(n.Describe().Returns(n.TString), func(a *n.Args) (any, error) {
	return os.Getwd()
})

var chdir = n.Function(n.Describe(n.Arg("dir", n.TString)), func(a *n.Args) (any, error) {
	return nil, os.Chdir(a.Name("dir").String())
})

var tempDir = n.Function(n.Describe().Returns(n.TString), func(a *n.Args) (any, error) {
	return os.TempDir(), nil
})
