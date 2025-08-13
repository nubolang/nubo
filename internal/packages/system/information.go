package system

import (
	"os"
	"runtime"

	"github.com/nubolang/nubo/native/n"
	"github.com/shirou/gopsutil/host"
	"golang.org/x/term"
)

var osName = n.Function(n.Describe().Returns(n.TString), func(a *n.Args) (any, error) {
	return runtime.GOOS, nil
})

var arch = n.Function(n.Describe().Returns(n.TString), func(a *n.Args) (any, error) {
	return runtime.GOARCH, nil
})

var hostname = n.Function(n.Describe().Returns(n.TString), func(a *n.Args) (any, error) {
	return os.Hostname()
})

var user = n.Function(n.Describe().Returns(n.TString), func(a *n.Args) (any, error) {
	return os.Getenv("USER"), nil
})

var cpu = n.Function(n.Describe().Returns(n.TInt), func(a *n.Args) (any, error) {
	return int64(runtime.NumCPU()), nil
})

var memoryTotal = n.Function(n.Describe().Returns(n.TInt), func(a *n.Args) (any, error) {
	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)
	return mem.Sys, nil
})

var memoryFree = n.Function(n.Describe().Returns(n.TInt), func(a *n.Args) (any, error) {
	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)
	return mem.Frees, nil
})

var uptime = n.Function(n.Describe().Returns(n.TInt), func(a *n.Args) (any, error) {
	return host.Uptime()
})

var isTTY = n.Function(n.Describe(n.Arg("fd", n.TInt)).Returns(n.TBool), func(a *n.Args) (any, error) {
	return term.IsTerminal(int(a.Name("fd").Value().(int64))), nil
})
