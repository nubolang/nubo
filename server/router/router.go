package router

import (
	"os"
	"path/filepath"
	"strings"
)

// Router represents a web router.
type Router struct {
	// root is the root directory of the router.
	root string

	// entries is a map of entries in the router.
	entries map[string]Entry
}

// New creates a new Router instance.
func New(root string) *Router {
	return &Router{
		root:    filepath.Clean(root),
		entries: make(map[string]Entry),
	}
}

// Reload reloads the router's entries.
func (r *Router) Reload() error {
	r.entries = make(map[string]Entry)

	return filepath.Walk(r.root, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}

		rel, err := filepath.Rel(r.root, path)
		if err != nil {
			return err
		}

		var routePath string
		exec := isExecutable(info.Name())

		if exec {
			routePath = "/" + strings.TrimSuffix(filepath.ToSlash(rel), filepath.Ext(rel))
		} else {
			routePath = "/" + filepath.ToSlash(rel)
		}

		if strings.HasSuffix(routePath, "/index") {
			routePath = strings.TrimSuffix(routePath, "/index")
		}
		if routePath == "" {
			routePath = "/"
		}

		// Break down the routePath into dynamic parts
		parts := make([]entryPart, 0)
		segments := strings.Split(routePath, "/")
		for _, segment := range segments {
			if segment == "" {
				continue
			}
			// If the segment is enclosed in square brackets, it's a parameter
			isParam := false
			name := segment
			if strings.HasPrefix(segment, "[") && strings.HasSuffix(segment, "]") {
				isParam = true
				name = segment[1 : len(segment)-1] // remove the brackets
			}

			parts = append(parts, entryPart{
				name:    name,
				isParam: isParam,
			})
		}

		// Store the Entry, which contains the parts
		r.entries[routePath] = Entry{
			FilePath:     path,
			Path:         rel,
			IsExecutable: exec,
			Parts:        parts,
		}

		return nil
	})
}

func (r *Router) Match(url string) (*Route, bool) {
	urlParts := make([]string, 0)
	segments := strings.Split(url, "/")
	for _, segment := range segments {
		if segment != "" {
			urlParts = append(urlParts, segment)
		}
	}

	var paramMatch *Route

	for _, entry := range r.entries {
		routeParts := entry.Parts
		if len(routeParts) != len(urlParts) {
			continue
		}

		match := true
		params := make(map[string]string, len(routeParts))
		exact := true

		for i, part := range routeParts {
			if part.isParam {
				params[part.name] = urlParts[i]
				exact = false
				continue
			}
			if part.name != urlParts[i] {
				match = false
				break
			}
		}

		if match {
			route := &Route{
				FilePath:     entry.FilePath,
				URLPath:      entry.Path,
				IsExecutable: entry.IsExecutable,
				Params:       params,
			}
			if exact {
				return route, true // exact match first
			}
			paramMatch = route // fallback if no exact match
		}
	}

	if paramMatch != nil {
		return paramMatch, true
	}
	return nil, false
}
