//go:build static && ios && !simulator

package manager

/*
#cgo ios LDFLAGS: ${SRCDIR}/../lib/ios/libdobby_bridge.a -framework CoreFoundation -framework Security -framework Foundation -framework Network -framework NetworkExtension -lc++
*/
import "C"
