package source

import (
	"io/fs"
	"strings"
	"time"
)

func NewModule(name, code string, files fs.FS) *Loader {
	return &Loader{
		files: &moduleFS{FS: files, code: code},
		label: name,
	}
}

type moduleFS struct {
	fs.FS
	code string
}

func (m *moduleFS) Open(name string) (fs.File, error) {
	if !fs.ValidPath(name) {
		return nil, &fs.PathError{Op: "open", Path: name, Err: fs.ErrInvalid}
	}

	if name == "package.nubo" && m.code != "" {
		return &moduleFile{Reader: strings.NewReader(m.code), size: int64(len(m.code))}, nil
	}

	if m.FS == nil {
		return nil, &fs.PathError{Op: "open", Path: name, Err: fs.ErrNotExist}
	}

	return m.FS.Open(name)
}

type moduleFile struct {
	*strings.Reader
	size int64
}

func (f *moduleFile) Close() error {
	return nil
}

func (f *moduleFile) Stat() (fs.FileInfo, error) {
	return moduleInfo{size: f.size}, nil
}

type moduleInfo struct {
	size int64
}

func (i moduleInfo) Name() string       { return "package.nubo" }
func (i moduleInfo) Size() int64        { return i.size }
func (i moduleInfo) Mode() fs.FileMode  { return 0444 }
func (i moduleInfo) ModTime() time.Time { return time.Time{} }
func (i moduleInfo) IsDir() bool        { return false }
func (i moduleInfo) Sys() any           { return nil }
