//go:build !cgo

// Stub for CGO_ENABLED=0 builds: the C++ core is unavailable and nothing
// references the kernels in this mode (pkg/vectormath falls back to
// pkg/vectormath/scalar). The stub keeps the package compilable so that
// `go build ./...` stays green without cgo.
package simd

// Enabled reports whether the C++ SIMD core is compiled into this binary.
const Enabled = false

// Variant returns the compiled kernel variant.
func Variant() string { return "none" }
