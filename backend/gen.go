// Package backend hosts only the go:generate directive; its name is kept
// distinct from the compiled packages under cmd/ and internal/ on purpose.
package backend

// tygo is an installed binary, not an import: go install github.com/gzuidhof/tygo@latest

//go:generate tygo generate
