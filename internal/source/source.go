package source

import (
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/nubolang/nubo/internal/ast/astnode"
	"github.com/nubolang/nubo/native"
)

type Loader struct {
	files fs.FS
	label string
}

func New(files fs.FS) *Loader {
	return &Loader{files: files}
}

func (l *Loader) Disk() bool {
	return l == nil || l.files == nil
}

func (l *Loader) Resolve(current, name string) (string, error) {
	if l.Disk() {
		if !filepath.IsAbs(name) {
			name = filepath.Join(filepath.Dir(current), name)
		}

		name = filepath.Clean(name)
		if filepath.Ext(name) == "" {
			name += ".nubo"
		}

		return name, nil
	}

	if name == "" || path.IsAbs(name) || strings.Contains(name, "\\") {
		return "", &fs.PathError{Op: "resolve", Path: name, Err: fs.ErrInvalid}
	}

	name = path.Join(path.Dir(filepath.ToSlash(current)), name)
	if !fs.ValidPath(name) {
		return "", &fs.PathError{Op: "resolve", Path: name, Err: fs.ErrInvalid}
	}

	if path.Ext(name) == "" {
		name += ".nubo"
	}

	return name, nil
}

func (l *Loader) Nodes(name string) ([]*astnode.Node, error) {
	if l.Disk() {
		return native.NodesFromFile(name, name)
	}

	file, err := l.files.Open(name)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	display := name
	if l.label != "" {
		display = l.label + "/" + filepath.ToSlash(name)
	}

	return native.NodesFromReader(file, display)
}

func (l *Loader) Stat(name string) (fs.FileInfo, error) {
	if l.Disk() {
		return os.Stat(name)
	}

	return fs.Stat(l.files, name)
}

func (l *Loader) Key(name string) string {
	if l.Disk() {
		if absolute, err := filepath.Abs(name); err == nil {
			return absolute
		}

		return filepath.Clean(name)
	}

	return path.Clean(filepath.ToSlash(name))
}

func (l *Loader) SameFile(first, second string) bool {
	if l.Key(first) == l.Key(second) {
		return true
	}

	if !l.Disk() {
		return false
	}

	firstInfo, firstErr := os.Stat(first)
	secondInfo, secondErr := os.Stat(second)

	return firstErr == nil && secondErr == nil && os.SameFile(firstInfo, secondInfo)
}

func (l *Loader) Workdir() (string, error) {
	if l.Disk() {
		return os.Getwd()
	}

	return ".", nil
}
