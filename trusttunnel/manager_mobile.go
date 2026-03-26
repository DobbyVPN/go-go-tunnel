//go:build android || ios

package trusttunnel

/*
#cgo CFLAGS: -I${SRCDIR}/dobby_bridge

// Android: Link the static library and Android's native logging/networking
#cgo android LDFLAGS: -L${SRCDIR}/lib/android/arm64-v8a -ldobby_bridge -llog -lm -lc++_shared

// iOS: Link the static library and Apple's Network Extension frameworks
#cgo ios LDFLAGS: -L${SRCDIR}/lib/ios -ldobby_bridge -framework Foundation -framework NetworkExtension
*/
import "C"
