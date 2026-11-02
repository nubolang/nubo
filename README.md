# Nubo ☁️

Nubo is a programming language for web apps, APIs, scripts, and tools. It combines readable syntax, typed values, HTML expressions, and a built-in HTTP server with file-based routing.

Write a script, serve a folder, or build a page with HTML directly in your code.

[Documentation](https://nubo.mrtn.vip) · [Downloads](https://github.com/nubolang/nubo/releases) · [Examples](example/v2)

> Nubo means “cloud” in Esperanto.

## Install

Download a binary for your platform from [GitHub Releases](https://github.com/nubolang/nubo/releases), extract it if needed, and place `nubo` somewhere on your `PATH`. On Windows, use `nubo.exe`.

Or install from source with Go 1.23.7 or newer:

```sh
go install github.com/nubolang/nubo/cmd/nubo@latest
```

For Go installations, make sure your Go binary directory is on your `PATH`—normally `$(go env GOPATH)/bin`, or your configured `GOBIN`.

Check the installation:

```sh
nubo --version
```

## Your first script

Save this as `hello.nubo`:

```rs
fn greet(name: string) string {
    return "Hello, " + name + "!"
}

println(greet("Nubo"))
```

Run it:

```sh
nubo hello.nubo
```

## A look at the language

Use `let` for variables and `const` for bindings that cannot be reassigned. Types can be inferred or written explicitly.

```rs
const name = "Nubo"
let total: int = 0
let numbers = [1, 2, 3]

for value in numbers {
    total = total + value
}

println(name, total)
```

Structs define data; `impl` blocks define their methods:

```rs
struct User {
    name: string
}

impl User {
    fn init(self: User, name: string) void {
        self.name = name
    }

    fn greet(self: User) string {
        return "Hello, " + self.name
    }
}

let user = User("Martin")
println(user.greet())
```

Capture an error with `catch`, then inspect the result:

```rs
catch err {
    panic("Something went wrong")
}

if !isNil(err) {
    println(err.message)
}
```

See the [language tests](tests) for examples of references, dictionaries, closures, interfaces, private members, and custom hooks.

## Build a web app

Create `public/index.nubo`:

```tsx
import response from "@server/response"

let name = "Nubo"

response.write(<main>
    <h1>Hello, {name}!</h1>
    <p>This page is written in Nubo.</p>
</main>)
```

Start the development server:

```sh
nubo serve public --dev --addr :3000
```

Open `http://localhost:3000`. HTML is a language value: you can store it in variables, return it from functions, and pass it to other functions. Interpolated text is escaped by default; `!{ ... }` inserts unescaped content.

Routes follow the directory structure:

| File | URL |
| --- | --- |
| `public/index.nubo` | `/` |
| `public/about.nubo` | `/about` |
| `public/api/users/index.nubo` | `/api/users` |
| `public/api/users/[id].nubo` | `/api/users/42`, with `id = "42"` |
| `public/style.css` | `/style.css` |

The server also supports serving a single `.nubo` file. Use `@server/request` to read request information and `@server/response` to write HTML, JSON, headers, cookies, and redirects.

For a JSON endpoint:

```tsx
import response from "@server/response"

response.json(dict[string, any] {
    message: "Hello from Nubo",
    ok: true
})
```

### Error pages and debugging

Add `public/error.nubo` to customize error responses:

```tsx
import response from "@server/response"
import error from "@server/error"

response.write(<main>
    <h1>{error.status}</h1>
    <p>{error.message}</p>
</main>)
```

If the error page fails, the server logs both failures and uses its built-in fallback. Development fallbacks show diagnostic details; production responses use generic messages for internal errors.

Use `--dev` for route reloading and development diagnostics. Add `--loglevel=DEBUG` for detailed runtime logs.

## Packages

Standard-library imports use `@std/`:

```nubo
import hash from "@std/hash"

println(hash.sha256("hello"))
```

The standard library includes modules for JSON, HTTP, SQL, files, time, hashing, components, reflection, and more.

Initialize a package and add a dependency:

```sh
nubo init
nubo get github.com/nubolang/color
```

Import it using its package name:

```nubo
import color from "@nubolang/color"

println(color.green("success"))
```

Run `nubo download` to download a project's declared dependencies. Local modules can be imported with relative paths, such as `import math from "./math"`.

## CLI reference

| Command | Purpose |
| --- | --- |
| `nubo main.nubo` | Run a script |
| `nubo serve public` | Serve a folder |
| `nubo serve main.nubo` | Serve a single file |
| `nubo init` | Initialize a package |
| `nubo get <repository>` | Add a dependency |
| `nubo download` | Download declared dependencies |
| `nubo prepare <file\|folder>` | Prepare parsed files for reuse |
| `nubo config` | Show the configuration path and current settings |

Use `nubo --help` or `nubo <command> --help` for options. The `format` command is currently unavailable.

## Working on Nubo

From a checkout of this repository:

```sh
make build
./bin/nubo hello.nubo

go test ./...
make smoketest
```

`make build` writes the CLI to `bin/nubo`. The smoke suite runs Nubo examples registered in [tests/run_all.nubo](tests/run_all.nubo).

Nubo is under active development. Language behavior, APIs, and tooling may change between versions.

## License

[Apache License 2.0](LICENSE).
