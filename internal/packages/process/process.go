package process

import (
	"bytes"
	"os/exec"

	"github.com/nubolang/nubo/internal/debug"
	"github.com/nubolang/nubo/language"
	"github.com/nubolang/nubo/native"
	"github.com/nubolang/nubo/native/n"
)

func NewProcess(dg *debug.Debug) language.Object {
	instance := n.NewPackage("process", dg)
	proto := instance.GetPrototype()

	proto.SetObject("run", native.NewTypedFunction(
		[]language.FnArg{
			&language.BasicFnArg{TypeVal: language.TypeString, NameVal: "cmd"},
			&language.BasicFnArg{TypeVal: language.TypeList, NameVal: "args", DefaultVal: language.NewList(nil, language.TypeAny, nil)},
		},
		language.TypeStructInstance,
		func(ctx native.FnCtx) (language.Object, error) {
			cmdObj, _ := ctx.Get("cmd")
			argsObj, _ := ctx.Get("args")

			cmdStr := cmdObj.Value().(string)
			args := []string{}
			for _, arg := range argsObj.Value().([]language.Object) {
				args = append(args, arg.String())
			}

			cmd := exec.Command(cmdStr, args...)
			var outBuf, errBuf bytes.Buffer
			cmd.Stdout = &outBuf
			cmd.Stderr = &errBuf

			err := cmd.Run()
			exitCode := 0
			if err != nil {
				if exitErr, ok := err.(*exec.ExitError); ok {
					exitCode = exitErr.ExitCode()
				} else {
					return nil, err
				}
			}

			result := language.NewStruct("@std/process:result", nil, nil)
			proto := result.GetPrototype()
			proto.SetObject("stdout", language.NewString(outBuf.String(), nil))
			proto.SetObject("stderr", language.NewString(errBuf.String(), nil))
			proto.SetObject("exit", language.NewInt(int64(exitCode), nil))

			return result, nil
		},
	))

	return instance
}
