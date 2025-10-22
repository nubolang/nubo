package os

import (
	"context"
	"os"

	"github.com/nubolang/nubo/internal/debug"
	"github.com/nubolang/nubo/internal/packages/time"
	"github.com/nubolang/nubo/language"
	"github.com/nubolang/nubo/native/n"
)

var dirEntry *language.Struct
var fileInfo *language.Struct

func NewOS(dg *debug.Debug) language.Object {
	instance := n.NewPackage("os", dg)
	proto := instance.GetPrototype()

	if fileInfo == nil {
		fileInfo = language.NewStruct("FileInfo", []language.StructField{
			{Name: "name", Type: n.TString},
			{Name: "isDir", Type: n.TBool},
			{Name: "size", Type: n.TInt},
			{Name: "mode", Type: n.TInt},
			{Name: "modTime", Type: time.GetInstance().Type()},
		}, dg)
	}

	if dirEntry == nil {
		dirEntry = language.NewStruct("DirEntry", []language.StructField{
			{Name: "name", Type: n.TString},
			{Name: "isDir", Type: n.TBool},
			{Name: "info", Type: language.NewFunctionType(fileInfo.Type())},
		}, dg)
	}

	ctx := context.Background()
	proto.SetObject(ctx, "readDir", n.Function(n.Describe(n.Arg("dir", n.TString)).Returns(n.TTList(dirEntry.Type())), readDir))

	return instance
}

func readDir(args *n.Args) (any, error) {
	dir := args.Name("dir")
	entries, err := os.ReadDir(dir.String())
	if err != nil {
		return nil, err
	}

	ctx := context.Background()

	var result = make([]language.Object, len(entries))
	for i, entry := range entries {
		inst, err := dirEntry.NewInstance()
		if err != nil {
			return nil, err
		}
		proto := inst.GetPrototype()
		proto.SetObject(ctx, "name", n.String(entry.Name(), dir.Debug()))
		proto.SetObject(ctx, "isDir", n.Bool(entry.IsDir(), dir.Debug()))

		proto.SetObject(ctx, "info", n.Function(n.Describe().Returns(fileInfo.Type()), func(a *n.Args) (any, error) {
			info, err := entry.Info()
			if err != nil {
				return nil, err
			}

			name := info.Name()
			isDir := info.IsDir()
			size := info.Size()
			mode := int64(info.Mode())
			modTime := info.ModTime()

			inst, err := fileInfo.NewInstance()
			if err != nil {
				return nil, err
			}

			proto := inst.GetPrototype()
			proto.SetObject(ctx, "name", n.String(name, dir.Debug()))
			proto.SetObject(ctx, "isDir", n.Bool(isDir, dir.Debug()))
			proto.SetObject(ctx, "size", n.Int64(size, dir.Debug()))
			proto.SetObject(ctx, "mode", n.Int64(mode, dir.Debug()))

			timeInst, err := time.NewInstance(modTime)
			if err != nil {
				return nil, err
			}
			proto.SetObject(ctx, "modTime", timeInst)

			return inst, nil
		}))

		result[i] = inst
	}

	return language.NewList(result, dirEntry.Type(), nil), nil
}
