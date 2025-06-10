package router

// Route represents a route that is matched.
type Route struct {
	// FilePath stores the path of the entry file.
	FilePath string
	// URLPath stores the URL path of the entry.
	URLPath string
	// IsExecutable marks whether the entry is executable.
	IsExecutable bool
	// Params stores URL parameters and other properties of the entry.
	Params map[string]string
}

// Entry is an URL in the folder
type Entry struct {
	// FilePath is the path with the root to the executable
	FilePath     string
	Path         string
	Parts        []entryPart
	IsExecutable bool
}

// entryPart is a part of the URL entry.
type entryPart struct {
	name    string
	isParam bool
}
