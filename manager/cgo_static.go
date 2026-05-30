//go:build static
// +build static

package manager

/*
// --- Static Linking Configuration ---
//
// Link against the bundled static library (.a) containing all dependencies.
//
// CRITICAL: We MUST use --whole-archive (or equivalent) to prevent the linker
// from stripping out C++ static constructors (like DNS provider factories).
// This was the root cause of the static segfaults/crashes!
//
// NOTE: Go's CGO explicitly blocks `-Wl,--whole-archive` for security reasons.
// To compile with `-tags static`, you MUST set the following environment variable:
//
// Linux/macOS: export CGO_LDFLAGS_ALLOW=".*"
// Windows (PS): $env:CGO_LDFLAGS_ALLOW=".*"
//
// Example: CGO_LDFLAGS_ALLOW=".*" go build -tags static ./examples/...

// macOS: Force load all symbols from the static library
#cgo darwin LDFLAGS: -L${SRCDIR}/../lib/macos -Wl,-force_load,${SRCDIR}/../lib/macos/libdobby_bridge.a -framework CoreFoundation -framework Security -framework Network -framework SystemConfiguration -framework Foundation -lresolv -lc++

// Linux: Use whole-archive to prevent stripping
#cgo linux,!android LDFLAGS: -L${SRCDIR}/../lib/linux -Wl,--whole-archive -ldobby_bridge -Wl,--no-whole-archive -lpthread -ldl -lresolv -lstdc++ -lm

// Windows: Use wholearchive (MinGW gcc format)
#cgo windows LDFLAGS: -L${SRCDIR}/../lib/windows -Wl,--whole-archive -ldobby_bridge -Wl,--no-whole-archive -lws2_32 -liphlpapi -lcrypt32 -lsecur32 -luserenv -lbcrypt -ladvapi32 -lfwpuclnt -lversion -lstdc++
*/
import "C"
