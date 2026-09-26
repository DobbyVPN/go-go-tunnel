//go:build static && android && amd64

package manager

/*
#cgo android,amd64 LDFLAGS: ${SRCDIR}/../lib/android/x86_64/libdobby_bridge.a -llog -lm -lc++_static -lc++abi
*/
import "C"
