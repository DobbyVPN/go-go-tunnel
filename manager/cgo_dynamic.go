//go:build !static
// +build !static

package manager

/*
// --- Dynamic Linking Configuration ---
//
// Link against shared libraries (.so/.dylib/.dll).
// This requires the dynamic library to be present at runtime.

// macOS: Link dynamic library and set rpath so it can be found at runtime
#cgo darwin LDFLAGS: -L${SRCDIR}/../lib/macos -ldobby_bridge -framework Foundation -framework NetworkExtension

// Linux: Link dynamic library and set rpath
#cgo linux,!android LDFLAGS: -L${SRCDIR}/../lib/linux -ldobby_bridge -lpthread -ldl -lc++ -lc++abi -lm

// Windows: Link against the DLL import library (dobby_bridge.lib / dobby_bridge.dll)
#cgo windows LDFLAGS: -L${SRCDIR}/../lib/windows -ldobby_bridge

// iOS: Link dynamic library and set rpath
#cgo ios,arm64 LDFLAGS: -L${SRCDIR}/../lib/ios -ldobby_bridge -framework Foundation -framework NetworkExtension

// Android: Link dynamic library for each ABI (architecture)
// ARM64
#cgo android,arm64 LDFLAGS: -L${SRCDIR}/../lib/android/arm64-v8a -ldobby_bridge -llog -lm -lc++_shared
*/
import "C"
