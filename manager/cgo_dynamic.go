//go:build !static
// +build !static

package manager

/*
// --- Dynamic Linking Configuration ---
//
// Link against shared libraries (.so/.dylib/.dll).
// This requires the dynamic library to be present at runtime.

// macOS: Link dynamic library and set rpath so it can be found at runtime
#cgo darwin LDFLAGS: -L${SRCDIR}/../lib/macos -ldobby_bridge -Wl,-rpath,@loader_path/../lib/macos -Wl,-rpath,@executable_path/../lib/macos

// Linux: Link dynamic library and set rpath
#cgo linux,!android LDFLAGS: -L${SRCDIR}/../lib/linux -ldobby_bridge -Wl,-rpath,$ORIGIN/../lib/linux

// Windows: Link against the DLL import library (dobby_bridge.lib / dobby_bridge.dll)
#cgo windows LDFLAGS: -L${SRCDIR}/../lib/windows -ldobby_bridge

// iOS: Link dynamic library and set rpath
#cgo ios,arm64 LDFLAGS: -L${SRCDIR}/../lib/ios -ldobby_bridge -Wl,-rpath,@executable_path/Frameworks -Wl,-rpath,@loader_path/Frameworks

// Android: Link dynamic library for each ABI (architecture)
// ARM64
#cgo android,arm64 LDFLAGS: -L${SRCDIR}/../lib/android/arm64-v8a -ldobby_bridge -Wl,-rpath,$ORIGIN/../lib/android/arm64-v8a

// ARM32
#cgo android,arm LDFLAGS: -L${SRCDIR}/../lib/android/armeabi-v7a -ldobby_bridge -Wl,-rpath,$ORIGIN/../lib/android/armeabi-v7a

// x86_64
#cgo android,amd64 LDFLAGS: -L${SRCDIR}/../lib/android/x86_64 -ldobby_bridge -Wl,-rpath,$ORIGIN/../lib/android/x86_64

// x86
#cgo android,386 LDFLAGS: -L${SRCDIR}/../lib/android/x86 -ldobby_bridge -Wl,-rpath,$ORIGIN/../lib/android/x86
*/
import "C"
