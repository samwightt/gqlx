# gqlx

`gqlx` is a tool for browsing `.graphql` files via the CLI. It contains useful commands that allow you (or whatever AI tool you prefer) to search through large schema files, analyze types, find how to query things, etc.

## Installation

To install, [first install `go`](https://go.dev/doc/install). On Mac, the quickest way to install Go is via `brew install go`.

After go installs, run:

```sh
go install github.com/samwightt/gqlx@main
```

Go will download and install the CLI for you. Last, make sure that your go binary path is in your PATH. Add the following to the end of your `~/.zshrc` if it's not already there (replacing `$HOME/go` with your GOPATH if it is different):

```sh
export PATH="$HOME/go/bin:$PATH"
```

Now you can use the CLI.

## Usage

```bash
# List all types in the schema
gqlx types -s schema.graphql

# Filter types by kind (type, enum, input, interface, union, scalar)
gqlx types --kind type --kind interface

# Find types that implement an interface
gqlx types --implements Node

# List fields on a specific type
gqlx fields User

# Find fields that return a specific type
gqlx fields --returns UserConnection

# Find fields with specific arguments
gqlx fields --has-arg first --has-arg after

# Show deprecated fields
gqlx fields --deprecated

# List arguments on a specific field
gqlx args Query.users

# Find all paths from Query to a type
gqlx paths User

# Find shortest path only
gqlx paths User --shortest

# List enum values
gqlx values StatusEnum

# Output as JSON for scripting
gqlx types --kind enum -f json | jq '.[].name'
```

### Global Flags

| Flag | Description |
|------|-------------|
| `-s, --schema` | Path to GraphQL schema file (default: `schema.graphql`) |
| `-f, --format` | Output format: `json`, `text`, `pretty` (default: `pretty` in terminal, `text` when piping) |
