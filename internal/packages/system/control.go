package system

import (
	"fmt"
	"os"
	"syscall"

	"github.com/nubolang/nubo/native/n"
)

var args = n.Function(n.Describe().Returns(n.TTList(n.TString)), func(a *n.Args) (any, error) {
	return os.Args, nil
})

var exit = n.Function(n.Describe(n.Arg("code", n.TInt)), func(a *n.Args) (any, error) {
	os.Exit(int(a.Name("code").Value().(int64)))
	return nil, nil
})

var abort = n.Function(n.Describe(n.Arg("message", n.TString)), func(a *n.Args) (any, error) {
	fmt.Fprintln(os.Stderr, "Fatal error:", a.Name("message").String())
	pid := os.Getpid()
	return nil, syscall.Kill(pid, syscall.SIGABRT)
})

var pid = n.Function(n.Describe().Returns(n.TInt), func(a *n.Args) (any, error) {
	pid := os.Getpid()
	return pid, nil
})

var kill = n.Function(n.Describe(n.Arg("pid", n.TInt)), func(a *n.Args) (any, error) {
	return nil, syscall.Kill(int(a.Name("pid").Value().(int64)), syscall.SIGABRT)
})
