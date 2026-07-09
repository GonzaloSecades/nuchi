//go:build tools

// Package tools pins developer tooling versions via the Go module system so
// generation commands are reproducible across machines and CI without a
// separate lockfile mechanism. This file is never compiled into the binary.
package tools

import (
	_ "github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen"
)
