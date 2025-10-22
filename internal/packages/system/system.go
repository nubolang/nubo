package system

import (
	"context"
	"os"

	"github.com/nubolang/nubo/internal/debug"
	"github.com/nubolang/nubo/language"
	"github.com/nubolang/nubo/native/n"
)

func NewSystem(dg *debug.Debug) language.Object {
	instance := n.NewPackage("system", dg)
	proto := instance.GetPrototype()

	ctx := context.Background()
	proto.SetObject(ctx, "stdin", n.Int(int(os.Stdin.Fd()), dg))
	proto.SetObject(ctx, "stdout", n.Int(int(os.Stdout.Fd()), dg))
	proto.SetObject(ctx, "stderr", n.Int(int(os.Stderr.Fd()), dg))

	// Program-control
	proto.SetObject(ctx, "args", args)
	proto.SetObject(ctx, "exit", exit)
	proto.SetObject(ctx, "abort", abort)
	proto.SetObject(ctx, "pid", pid)
	proto.SetObject(ctx, "kill", kill)

	// System information
	proto.SetObject(ctx, "osName", osName)
	proto.SetObject(ctx, "arch", arch)
	proto.SetObject(ctx, "hostname", hostname)
	proto.SetObject(ctx, "user", user)
	proto.SetObject(ctx, "cpu", cpu)
	proto.SetObject(ctx, "memoryTotal", memoryTotal)
	proto.SetObject(ctx, "memoryFree", memoryFree)
	proto.SetObject(ctx, "uptime", uptime)
	proto.SetObject(ctx, "isTTY", isTTY)

	// Filesystem
	proto.SetObject(ctx, "cwd", cwd)
	proto.SetObject(ctx, "chdir", chdir)
	proto.SetObject(ctx, "tempDir", tempDir)

	return instance
}
