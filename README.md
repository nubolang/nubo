# Nubo ☁️

Nubo is a new real-time programming language for web.
Nubo is designed to be easy to learn and use for creating interactive web applications.

## Installation

To Installation Nubo, you have to install Go first. You can download it from [here](https://golang.org/dl/).

To Installation Nubo, you can simply use the following command:

```bash
go install github.com/nubolang/nubo/cmd/nubo@latest
```

## Our Future Concepts

We believe in simplicity and efficiency. Nubo is designed with `pub` and `sub` keywords for real-time communication. This language does not provide `async` or `await` keywords, as it is designed to be
as simple as possible. With these built-in keywords, developers can easily `subscribe` to events (even on client-side) and `publish` events to other clients.

Nubo also has built-in support for `HTML` templates. After any expression, developers can pass `HTML` contents without messing with string literals. By default, Nubo prevents text from being vulnerable to XSS attacks. When embedding other HTML code inside a component, you should prefix it with `@` so that our parser doesn't escape it.
Here is a short example:

```jsx
let html = <div>Hello, World!</div>

return <body>
  @{html}
</body>
```

With this features, developers can easily create interactive web applications with ease without touching javascript.

**Then what about CSS?**

With our concept, Nubo will contains a set of built-in styled `HTML` components to make development faster then ever. Creating a full feature-rich client is hard, but with Nubo you can write something like this:
```jsx
import styled from "@std/styled"

let user = styled.User("Martin")
let dashboard = <styled.AdminDashboard context="index" :user="user">
  <styled.H1>Hello, {user.name}!</styled.H1>
<styled.AdminDashboard>
```

## Extend how you like

Nubo will provide a feature to install it as a Go module via `go get`. With this, developers will be
able to create their custom libraries, define their own Go functions and map them to Nubo's interpreter via
our `native` module.
