package system

import (
	"os"

	"github.com/nubolang/nubo/internal/debug"
	"github.com/nubolang/nubo/language"
	"github.com/nubolang/nubo/native/n"
)

func NewSystem(dg *debug.Debug) language.Object {
	instance := n.NewPackage("system", dg)
	proto := instance.GetPrototype()

	proto.SetObject("stdin", n.Int(int(os.Stdin.Fd()), dg))
	proto.SetObject("stdout", n.Int(int(os.Stdout.Fd()), dg))
	proto.SetObject("stderr", n.Int(int(os.Stderr.Fd()), dg))

	// Program-control
	proto.SetObject("args", args)
	proto.SetObject("exit", exit)
	proto.SetObject("abort", abort)
	proto.SetObject("pid", pid)
	proto.SetObject("kill", kill)

	// System information
	proto.SetObject("osName", osName)
	proto.SetObject("arch", arch)
	proto.SetObject("hostname", hostname)
	proto.SetObject("user", user)
	proto.SetObject("cpu", cpu)
	proto.SetObject("memoryTotal", memoryTotal)
	proto.SetObject("memoryFree", memoryFree)
	proto.SetObject("uptime", uptime)
	proto.SetObject("isTTY", isTTY)

	// Filesystem
	proto.SetObject("cwd", cwd)
	proto.SetObject("chdir", chdir)
	proto.SetObject("tempDir", tempDir)

	return instance
}
