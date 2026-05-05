package dotfolder

import (
	"context"
	"crypto/sha256"
	"encoding/gob"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/nubolang/nubo/internal/ast"
	"github.com/nubolang/nubo/internal/ast/astnode"
	"github.com/nubolang/nubo/internal/lexer"
)

func init() {
	gob.Register(&astnode.Node{})
	gob.Register(&astnode.ForValue{})
}

const dirHashManifestFile = "dirhash.json"

var (
	manifestMu      sync.Mutex
	manifestRootDir string
	manifestLoaded  bool
	manifestData    map[string]string
	hashCache       map[string]string
)

func PrepareFiles(target string, sameDir bool) error {
	target = filepath.Clean(target)
	info, err := os.Stat(target)
	if err != nil {
		return err
	}

	if info.IsDir() {
		if err := filepath.WalkDir(target, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() || filepath.Ext(path) != ".nubo" {
				return nil
			}
			return prepareOne(path, sameDir)
		}); err != nil {
			return err
		}

		// Dir hash trust is updated only when preparing a whole directory.
		if !sameDir {
			_ = updatePreparedDirHash(target)
		}

		return nil
	}

	if filepath.Ext(target) != ".nubo" {
		return nil
	}

	return prepareOne(target, sameDir)
}

func prepareOne(path string, sameDir bool) error {
	path = filepath.Clean(path)

	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	lx, err := lexer.New(file, path)
	if err != nil {
		return err
	}
	tokens, err := lx.Parse()
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	parser := ast.New(ctx)
	nodes, err := parser.Parse(tokens)
	if err != nil {
		return err
	}

	return writePrepared(path, nodes, sameDir)
}

func HasPrepared(file string) ([]*astnode.Node, bool) {
	file = filepath.Clean(file)
	preparedPath := preparedPathFor(file, false)
	if preparedPath == "" {
		return nil, false
	}

	// fallback path (same dir)
	if _, err := os.Stat(preparedPath); os.IsNotExist(err) {
		// fallback path (legacy file-local cache)
		dir := filepath.Dir(file)
		base := strings.TrimSuffix(filepath.Base(file), ".nubo") + ".nuboc"
		preparedPath = filepath.Join(dir, RootFolderName, PreparedFolderName, base)
		if _, err := os.Stat(preparedPath); os.IsNotExist(err) {
			// fallback path (same dir)
			preparedPath = filepath.Join(dir, base)
		}
	}

	if !isPreparedDirHashTrusted(file) {
		preparedInfo, err := os.Stat(preparedPath)
		if err != nil {
			return nil, false
		}
		originalInfo, err := os.Stat(file)
		if err != nil {
			return nil, false
		}
		if originalInfo.ModTime().After(preparedInfo.ModTime()) {
			return nil, false
		}
	}

	f, err := os.Open(preparedPath)
	if err != nil {
		return nil, false
	}
	defer f.Close()

	var nodes []*astnode.Node
	if err := gob.NewDecoder(f).Decode(&nodes); err != nil {
		return nil, false
	}

	return nodes, true
}

func SavePrepared(file string, nodes []*astnode.Node) error {
	file = filepath.Clean(file)
	return writePrepared(file, nodes, false)
}

func writePrepared(file string, nodes []*astnode.Node, sameDir bool) error {
	dest := preparedPathFor(file, sameDir)
	if dest == "" {
		return nil
	}

	if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
		return err
	}

	out, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer out.Close()

	return gob.NewEncoder(out).Encode(nodes)
}

func preparedPathFor(file string, sameDir bool) string {
	base := strings.TrimSuffix(filepath.Base(file), ".nubo") + ".nuboc"
	if sameDir {
		return filepath.Join(filepath.Dir(file), base)
	}

	wd, err := os.Getwd()
	if err != nil {
		return ""
	}

	cacheRoot := filepath.Join(wd, RootFolderName, PreparedFolderName)

	absFile, err := filepath.Abs(file)
	if err != nil {
		return filepath.Join(cacheRoot, base)
	}
	absWd, err := filepath.Abs(wd)
	if err != nil {
		return filepath.Join(cacheRoot, base)
	}

	rel, err := filepath.Rel(absWd, absFile)
	if err == nil && rel != "." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) && rel != ".." {
		return filepath.Join(cacheRoot, strings.TrimSuffix(rel, ".nubo")+".nuboc")
	}

	h := fnv.New32a()
	_, _ = h.Write([]byte(absFile))
	return filepath.Join(cacheRoot, "external", fmt.Sprintf("%08x.nuboc", h.Sum32()))
}

func updatePreparedDirHash(dir string) error {
	wd, err := os.Getwd()
	if err != nil {
		return err
	}
	absWd, err := filepath.Abs(wd)
	if err != nil {
		return err
	}
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return err
	}

	rel, err := filepath.Rel(absWd, absDir)
	if err != nil {
		return nil
	}
	if rel == "." {
		rel = "."
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		// only track dirs inside current project
		return nil
	}
	rel = filepath.ToSlash(filepath.Clean(rel))

	sum, err := hashNuboDir(absDir)
	if err != nil {
		return err
	}

	manifest, err := loadDirHashManifest(absWd)
	if err != nil {
		return err
	}
	manifest[rel] = sum
	return saveDirHashManifest(absWd, manifest)
}

func isPreparedDirHashTrusted(file string) bool {
	wd, err := os.Getwd()
	if err != nil {
		return false
	}
	absWd, err := filepath.Abs(wd)
	if err != nil {
		return false
	}
	absFile, err := filepath.Abs(file)
	if err != nil {
		return false
	}

	relFile, err := filepath.Rel(absWd, absFile)
	if err != nil || relFile == ".." || strings.HasPrefix(relFile, ".."+string(filepath.Separator)) {
		return false
	}

	manifest, err := loadDirHashManifest(absWd)
	if err != nil || len(manifest) == 0 {
		return false
	}

	dir := filepath.ToSlash(filepath.Clean(filepath.Dir(relFile)))
	if dir == "." || dir == "" {
		dir = "."
	}

	for {
		expected, ok := manifest[dir]
		if ok {
			actual, err := hashNuboDirCached(absWd, dir)
			if err != nil {
				return false
			}
			return actual == expected
		}

		if dir == "." {
			break
		}

		next := filepath.ToSlash(filepath.Dir(dir))
		if next == "" {
			next = "."
		}
		if next == dir {
			break
		}
		dir = next
	}

	return false
}

func hashNuboDirCached(absWd, relDir string) (string, error) {
	manifestMu.Lock()
	defer manifestMu.Unlock()

	if hashCache == nil {
		hashCache = make(map[string]string)
	}

	key := absWd + "::" + relDir
	if v, ok := hashCache[key]; ok {
		return v, nil
	}

	target := absWd
	if relDir != "." {
		target = filepath.Join(absWd, filepath.FromSlash(relDir))
	}
	sum, err := hashNuboDir(target)
	if err != nil {
		return "", err
	}
	hashCache[key] = sum
	return sum, nil
}

func hashNuboDir(absDir string) (string, error) {
	type entry struct {
		rel     string
		size    int64
		modUnix int64
	}

	var entries []entry
	err := filepath.WalkDir(absDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			name := d.Name()
			if name == ".git" || name == RootFolderName {
				return filepath.SkipDir
			}
			return nil
		}

		if filepath.Ext(path) != ".nubo" {
			return nil
		}

		info, err := d.Info()
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(absDir, path)
		if err != nil {
			return err
		}

		entries = append(entries, entry{
			rel:     filepath.ToSlash(rel),
			size:    info.Size(),
			modUnix: info.ModTime().UnixNano(),
		})
		return nil
	})
	if err != nil {
		return "", err
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].rel < entries[j].rel
	})

	hasher := sha256.New()
	for _, e := range entries {
		_, _ = io.WriteString(hasher, e.rel)
		_, _ = io.WriteString(hasher, "|")
		_, _ = io.WriteString(hasher, fmt.Sprintf("%d|%d\n", e.size, e.modUnix))
	}

	return hex.EncodeToString(hasher.Sum(nil)), nil
}

func loadDirHashManifest(absWd string) (map[string]string, error) {
	manifestMu.Lock()
	defer manifestMu.Unlock()

	if manifestLoaded && manifestRootDir == absWd && manifestData != nil {
		out := make(map[string]string, len(manifestData))
		for k, v := range manifestData {
			out[k] = v
		}
		return out, nil
	}

	path := filepath.Join(absWd, RootFolderName, PreparedFolderName, dirHashManifestFile)
	data := make(map[string]string)

	f, err := os.Open(path)
	if err == nil {
		defer f.Close()
		_ = json.NewDecoder(f).Decode(&data)
	} else if !os.IsNotExist(err) {
		return nil, err
	}

	manifestRootDir = absWd
	manifestLoaded = true
	manifestData = data
	if hashCache == nil {
		hashCache = make(map[string]string)
	}

	out := make(map[string]string, len(data))
	for k, v := range data {
		out[k] = v
	}
	return out, nil
}

func saveDirHashManifest(absWd string, manifest map[string]string) error {
	path := filepath.Join(absWd, RootFolderName, PreparedFolderName, dirHashManifestFile)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}

	tmp := path + ".tmp"
	f, err := os.Create(tmp)
	if err != nil {
		return err
	}

	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	if err := enc.Encode(manifest); err != nil {
		f.Close()
		_ = os.Remove(tmp)
		return err
	}
	if err := f.Close(); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return err
	}

	manifestMu.Lock()
	manifestRootDir = absWd
	manifestLoaded = true
	manifestData = make(map[string]string, len(manifest))
	for k, v := range manifest {
		manifestData[k] = v
	}
	manifestMu.Unlock()

	return nil
}
