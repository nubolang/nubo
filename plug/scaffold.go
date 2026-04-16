package plug

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/charmbracelet/huh"
	"gopkg.in/yaml.v3"
)

// mainTemplateStdio is the starter plugin using stdio transport.
const mainTemplateStdio = `package main

import "github.com/nubolang/nubo/plug"

func main() {
	app := plug.Create()

	// Register your handlers here.
	app.Handler("ping", func(ctx *plug.Ctx) error {
		var req plug.Map
		if err := ctx.Bind(&req); err != nil {
			return ctx.Fail(err)
		}
		return ctx.Send(plug.Map{"status": "pong"})
	})

	app.Start()
}
`

// mainTemplateTCP is the starter plugin using TCP transport (no auth).
const mainTemplateTCP = `package main

import (
	"log"

	"github.com/nubolang/nubo/plug"
)

func main() {
	// Bind on :0 to let the OS pick a free port.
	// The Manager discovers the actual address automatically.
	app, err := plug.CreateTCP(":0")
	if err != nil {
		log.Fatal(err)
	}

	// Register your handlers here.
	app.Handler("ping", func(ctx *plug.Ctx) error {
		var req plug.Map
		if err := ctx.Bind(&req); err != nil {
			return ctx.Fail(err)
		}
		return ctx.Send(plug.Map{"status": "pong"})
	})

	app.Start()
}
`

// mainTemplateTCPAuth is the starter plugin using TCP transport with a custom
// auth validator.
const mainTemplateTCPAuth = `package main

import (
	"log"
	"os"

	"github.com/nubolang/nubo/plug"
)

func main() {
	// Bind on :0 to let the OS pick a free port.
	// The Manager discovers the actual address automatically.
	//
	// WithAuthValidator lets you implement any token-checking logic you need:
	// environment variables, files, HMAC, external services, etc.
	app, err := plug.CreateTCP(":0", plug.WithAuthValidator(func(token string) bool {
		secret := os.Getenv("PLUGIN_SECRET")
		return secret != "" && token == secret
	}))
	if err != nil {
		log.Fatal(err)
	}

	// Register your handlers here.
	app.Handler("ping", func(ctx *plug.Ctx) error {
		var req plug.Map
		if err := ctx.Bind(&req); err != nil {
			return ctx.Fail(err)
		}
		return ctx.Send(plug.Map{"status": "pong"})
	})

	app.Start()
}
`

// gitignoreContent ignores all compiled binaries in bin/ but keeps the file itself.
const gitignoreContent = `*
!.gitignore
`

// supportedArchitectures lists the OS values the scaffold knows about.
var supportedArchitectures = []string{
	"linux",
	"windows",
	"darwin",
}

// Scaffold interactively creates a new plugin project at path.
func Scaffold(path string) error {
	if err := os.MkdirAll(path, 0755); err != nil {
		return fmt.Errorf("scaffold: mkdir %s: %w", path, err)
	}

	backendDir := filepath.Join(path, "backend")
	if err := os.MkdirAll(backendDir, 0755); err != nil {
		return fmt.Errorf("scaffold: mkdir backend: %w", err)
	}

	moduleName, err := promptText("Go module name")
	if err != nil {
		return err
	}

	cgoChoice, err := promptSelect("CGO enabled", []string{"false", "true"})
	if err != nil {
		return err
	}

	archs, err := promptMultiSelect("Target operating systems", supportedArchitectures)
	if err != nil {
		return err
	}

	transportMode, err := promptSelect("Transport mode", []string{"stdio", "tcp"})
	if err != nil {
		return err
	}

	mainSrc := mainTemplateStdio
	if transportMode == "tcp" {
		authChoice, err := promptSelect("Require token authentication?", []string{"no", "yes"})
		if err != nil {
			return err
		}
		if authChoice == "yes" {
			mainSrc = mainTemplateTCPAuth
		} else {
			mainSrc = mainTemplateTCP
		}
	}

	cfg := PlugConfig{
		Plugin: PluginMeta{
			CGO:          cgoChoice == "true",
			Source:       ".",
			Cmd:          "go build -o {dest} {src}",
			Binary:       "./bin/plug",
			Architecture: archs,
			Transport: TransportConfig{
				Mode: transportMode,
			},
		},
	}

	yamlData, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}

	backendFiles := []struct{ file, content string }{
		{"_plug.yaml", string(yamlData)},
		{"main.go", mainSrc},
	}
	for _, w := range backendFiles {
		if err := os.WriteFile(filepath.Join(backendDir, w.file), []byte(w.content), 0644); err != nil {
			return fmt.Errorf("scaffold: write backend/%s: %w", w.file, err)
		}
	}

	binDir := filepath.Join(backendDir, "bin")
	if err := os.MkdirAll(binDir, 0755); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(binDir, ".gitignore"), []byte(gitignoreContent), 0644); err != nil {
		return fmt.Errorf("scaffold: write backend/bin/.gitignore: %w", err)
	}

	nuboContent := buildNuboExample(transportMode)
	if err := os.WriteFile(filepath.Join(path, "example.nubo"), []byte(nuboContent), 0644); err != nil {
		return fmt.Errorf("scaffold: write example.nubo: %w", err)
	}

	fmt.Printf("\nInitialising Go module %q in backend/ ...\n", moduleName)
	if err := runCmd(backendDir, "go", "mod", "init", moduleName); err != nil {
		return fmt.Errorf("scaffold: go mod init: %w", err)
	}

	fmt.Println("Downloading module dependencies ...")
	if err := runCmd(backendDir, "go", "mod", "download"); err != nil {
		fmt.Printf("warning: go mod download: %v\n", err)
	}

	fmt.Printf("\nPlugin scaffolded at %s\n", path)
	fmt.Println("Next: edit backend/main.go, then load the plugin in example.nubo (or rename it).")
	return nil
}

// buildNuboExample returns the content of the example.nubo file tailored to
// the chosen transport mode.
func buildNuboExample(mode string) string {
	callPing := `const result = app.send("ping")
const status = result.status
println("ping result:", string(status.map(byte)))
`

	if mode == "tcp" {
		return `import plug from "@std/plug"

const app = plug.require(mode: "tcp", path: __dir__ + "/backend")

` + callPing
	}
	return `import plug from "@std/plug"

const app = plug.require(mode: "stdio", path: __dir__ + "/backend")

` + callPing
}

// runCmd runs a command in dir, streaming stdout/stderr to the terminal.
func runCmd(dir string, name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// promptText prompts the user for a single-line string.
func promptText(label string) (string, error) {
	var value string

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title(label).
				Value(&value),
		),
	)

	if err := form.Run(); err != nil {
		return "", err
	}

	return value, nil
}

// promptSelect shows a single-choice list.
func promptSelect(label string, items []string) (string, error) {
	if len(items) == 0 {
		return "", fmt.Errorf("no items provided")
	}

	var value string
	options := make([]huh.Option[string], 0, len(items))
	for _, item := range items {
		options = append(options, huh.NewOption(item, item))
	}

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title(label).
				Options(options...).
				Value(&value),
		),
	)

	if err := form.Run(); err != nil {
		return "", err
	}

	return value, nil
}

// promptMultiSelect shows a toggle list where the user can select multiple
func promptMultiSelect(label string, items []string) ([]string, error) {
	if len(items) == 0 {
		return nil, fmt.Errorf("no items provided")
	}

	options := make([]huh.Option[string], 0, len(items))
	for _, item := range items {
		options = append(options, huh.NewOption(item, item))
	}

	var selected []string

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewMultiSelect[string]().
				Title(label).
				Options(options...).
				Value(&selected),
		),
	)

	if err := form.Run(); err != nil {
		return nil, err
	}

	if len(selected) == 0 {
		return nil, fmt.Errorf("please select at least one item")
	}

	return selected, nil
}
