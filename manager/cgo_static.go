//go:build static
// +build static

package manager

/*
// --- Static Linking Configuration ---
//
// Link against the bundled static library (.a) containing all dependencies.

// iOS
#cgo ios LDFLAGS: ${SRCDIR}/../lib/ios/libdobby_bridge.a -framework CoreFoundation -framework Security -framework Foundation -framework Network -framework NetworkExtension -lc++

// macOS
#cgo darwin,!ios LDFLAGS: ${SRCDIR}/../lib/macos/libdobby_bridge.a -framework CoreFoundation -framework Security -framework Foundation -framework Network -framework NetworkExtension -lc++

// Linux: Use whole-archive to prevent stripping
#cgo linux,!android LDFLAGS: ${SRCDIR}/../lib/linux/libdobby_bridge.a -lpthread -ldl -lc++ -lc++abi -lm

#cgo windows LDFLAGS: ${SRCDIR}/../lib/windows/libdobby_bridge.a
*/
import "C"
