package io

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/nubolang/nubo/language"
	"github.com/nubolang/nubo/native"
	"golang.org/x/text/encoding"
	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/encoding/unicode"
	"golang.org/x/text/transform"
)

func openFn(ctx native.FnCtx) (language.Object, error) {
	fileName, err := ctx.Get("file")
	if err != nil {
		return nil, err
	}
	encodingName, err := ctx.Get("encoding")
	if err != nil {
		return nil, err
	}

	rawMode, err := ctx.Get("mode")
	if err != nil {
		return nil, err
	}
	modeStr := rawMode.String()

	flags, err := modeStringToFlags(modeStr)
	if err != nil {
		return nil, err
	}

	rawPerm, err := ctx.Get("perm")
	if err != nil {
		return nil, err
	}
	perm := os.FileMode(rawPerm.Value().(int64))

	fileRealPath := fileName.String()
	if !filepath.IsAbs(fileRealPath) && fileName.Debug() != nil {
		fileRealPath = filepath.Join(filepath.Dir(fileName.Debug().File), fileRealPath)
	}

	enc, err := getEncoding(encodingName.String())
	if err != nil {
		return nil, err
	}

	file, err := os.OpenFile(fileRealPath, flags, perm)
	if err != nil {
		return nil, err
	}

	var (
		reader io.Reader = file
		writer io.Writer = nil
	)

	if enc != encoding.Nop {
		reader = transform.NewReader(file, enc.NewDecoder())
	}

	if flags&(os.O_WRONLY|os.O_RDWR|os.O_APPEND) != 0 {
		writer = file
	}

	return NewIOStream(reader, writer), nil
}

func getEncoding(name string) (encoding.Encoding, error) {
	switch strings.ToLower(name) {
	case "utf-8":
		return encoding.Nop, nil
	case "utf-16":
		return unicode.UTF16(unicode.LittleEndian, unicode.ExpectBOM), nil
	case "utf-16le":
		return unicode.UTF16(unicode.LittleEndian, unicode.IgnoreBOM), nil
	case "utf-16be":
		return unicode.UTF16(unicode.BigEndian, unicode.IgnoreBOM), nil
	case "latin1", "iso-8859-1":
		return charmap.ISO8859_1, nil
	case "windows-1252", "cp1252":
		return charmap.Windows1252, nil
	default:
		return nil, fmt.Errorf("unsupported encoding: %s", name)
	}
}

func modeStringToFlags(modeStr string) (int, error) {
	switch modeStr {
	case "r": // read-only
		return os.O_RDONLY, nil
	case "r+", "rw": // read/write, must exist, no truncation
		return os.O_RDWR, nil
	case "w": // write-only, create/truncate
		return os.O_WRONLY | os.O_CREATE | os.O_TRUNC, nil
	case "w+", "rw+": // read/write, create/truncate
		return os.O_RDWR | os.O_CREATE | os.O_TRUNC, nil
	case "a": // write-only, append/create
		return os.O_WRONLY | os.O_CREATE | os.O_APPEND, nil
	case "a+": // read/write, append/create
		return os.O_RDWR | os.O_CREATE | os.O_APPEND, nil
	default:
		return 0, fmt.Errorf("invalid mode string: %s, supported: r, r+, rw, w, w+, rw+, a, a+", modeStr)
	}
}
