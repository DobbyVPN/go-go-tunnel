//go:build static
// +build static

package manager

/*
// --- Static Linking Configuration ---
//
// Link against the bundled static library (.a) containing all dependencies.

// macOS: Force load all symbols from the static library
#cgo darwin LDFLAGS: -L${SRCDIR}/../lib/macos/libdobby_bridge.a -framework CoreFoundation -framework Security

// Linux: Use whole-archive to prevent stripping
#cgo linux,!android LDFLAGS: -L${SRCDIR}/../lib/linux/libdobby_bridge.a -lpthread -ldl -lc++ -lc++abi -lm

#cgo windows LDFLAGS: -L${SRCDIR}/../lib/windows/libdobby_bridge.a
*/
import "C"
