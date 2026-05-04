# Nubo ☁️

Nubo is a small programming language for building web apps, APIs, scripts, and tools.

It is designed to make server-side web development feel simple: write normal Nubo code, return HTML values, organize routes as files, and use a standard library instead of wiring everything from scratch.

> Nubo means “cloud” in Esperanto.

## Why Nubo?

Nubo is built around a few simple ideas:

- **Readable syntax** that is easy to learn.
- **HTML as a language value**, not just strings.
- **File-based routing** for web apps.
- **Structs and methods** for clean application code.
- **Typed values** without making simple code noisy.
- **Standard libraries** for common web, system, database, component, and reflection tasks.
- **Package imports** using simple names like `@nubolang/color`.

## A Small Taste

```nubo
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

## HTML Without String Templates

Nubo can write HTML directly:

```nubo
let name = "Nubo"

let page = <main>
    <h1>Hello, { name }</h1>
    <p>Build pages without turning HTML into strings.</p>
</main>
```

Interpolated values are escaped by default. Trusted HTML can be inserted with `!{ ... }`.

## Web Apps

Nubo can serve a folder of `.nubo` files:

```bash
nubo serve public
```

Example route:

```nubo
import { write: writeResponse } from "@server/response"

writeResponse(<main>
    <h1>Hello from Nubo</h1>
</main>)
```

Example structure:

```txt
public/
  index.nubo
  about.nubo
  api/
    users/
      index.nubo
      [id].nubo
```

## Packages

Install a package:

```bash
nubo get github.com/nubolang/color
```

Use it:

```nubo
import color from "@nubolang/color"

println(color.green("success"))
```

## Installation

Install Go first, then install Nubo:

```bash
go install github.com/nubolang/nubo/cmd/nubo@latest
```

Check it:

```bash
nubo --version
```

## CLI

```bash
nubo file.nubo
nubo serve public
nubo init
nubo get github.com/user/package
nubo download
nubo config
```

## Learn More

Read the docs and guides:

https://nubo.mrtn.vip

## Status

Nubo is under active development. The language, standard libraries, and tooling may change as the project grows.
