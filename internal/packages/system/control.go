package system

import (
	"fmt"
	"os"

	"github.com/nubolang/nubo/language"
	"github.com/nubolang/nubo/native/n"
)

var args = n.Function(n.Describe().Returns(n.TTList(n.TString)), func(a *n.Args) (any, error) {
	var args = make([]any, len(os.Args))
	for i, arg := range os.Args {
		args[i] = language.NewString(arg, nil)
	}
	return n.List(args, nil)
})

var exit = n.Function(n.Describe(n.Arg("code", n.TInt)), func(a *n.Args) (any, error) {
	os.Exit(int(a.Name("code").Value().(int64)))
	return nil, nil
})

var abort = n.Function(
    n.Describe(n.Arg("message", n.TString)),
    func(a *n.Args) (any, error) {
        fmt.Fprintln(os.Stderr, "Fatal error:", a.Name("message").String())
        pid := os.Getpid()
        p, err := os.FindProcess(pid)
        if err != nil {
            return nil, err
        }
        return nil, p.Kill()
    },
)

var pid = n.Function(n.Describe().Returns(n.TInt), func(a *n.Args) (any, error) {
	pid := os.Getpid()
	return pid, nil
})

var kill = n.Function(
    n.Describe(n.Arg("pid", n.TInt)),
    func(a *n.Args) (any, error) {
        pid := int(a.Name("pid").Value().(int64))
        p, err := os.FindProcess(pid)
        if err != nil {
            return nil, err
        }
        return nil, p.Kill()
    },
)
