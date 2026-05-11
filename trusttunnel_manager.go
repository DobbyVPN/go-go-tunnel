package main

/*
#cgo CFLAGS: -I${SRCDIR}/dobby_bridge

// --- OS-Specific Linking ---
// Windows expects a file named dobby_bridge.dll and dobby_bridge.lib in the same directory
#cgo windows LDFLAGS: -L${SRCDIR}/lib/windows -ldobby_bridge

// Linux expects a file named dobby_bridge.so
#cgo linux LDFLAGS: -L${SRCDIR}/lib/linux -ldobby_bridge -lpthread -ldl -lc++ -lc++abi -lm

// macOS expects a file named dobby_bridge.dylib
#cgo darwin LDFLAGS: -L${SRCDIR}/lib/macos -ldobby_bridge -framework CoreFoundation -framework Security

// Android: Link the static library and Android's native logging/networking
#cgo android LDFLAGS: -L${SRCDIR}/lib/android/arm64-v8a -ldobby_bridge -llog -lm -lc++_shared

// iOS: Link the static library and Apple's Network Extension frameworks
#cgo ios LDFLAGS: -L${SRCDIR}/lib/ios -ldobby_bridge -framework Foundation -framework NetworkExtension

#include <stdlib.h>
#include "dobby_bridge/dobby_bridge_common.h"

// 1. Define the C-Callback type
typedef int (*protect_cb_t)(int fd);

// 2. Helper to execute the callback safely in C
static inline int execute_protect_cb(protect_cb_t cb, int fd) {
    if (cb == NULL) return 0;
    return cb(fd);
}

// C "Gateway" functions
extern void c_state_changed(void* arg, int state);
extern void c_log_message(int level, const char* msg);
extern int c_protect_cb(int fd);
*/
import "C"
import (
	"log"
	"unsafe"
)

// Global reference to the mobile protect callback pointer (for Android/iOS)
var globalProtectCallback C.protect_cb_t

// SetMobileProtectCallback is called from your exported wrapper for mobile platforms
func SetMobileProtectCallback(ptr unsafe.Pointer) {
	globalProtectCallback = (C.protect_cb_t)(ptr)
}

//export go_state_changed
func go_state_changed(arg unsafe.Pointer, state C.int) {
	log.Printf("[TrustTunnel] State changed to: %d", int(state))
}

//export go_log_message
func go_log_message(level C.int, msg *C.char) {
	goMsg := C.GoString(msg)

	// ag::LogLevel mapping from logger.h: 0=ERROR, 1=WARN, 2=INFO, 3=DEBUG, 4=TRACE
	switch int(level) {
	case 0:
		log.Printf("[TrustTunnel Core] ERROR: %s", goMsg)
	case 1:
		log.Printf("[TrustTunnel Core] WARN: %s", goMsg)
	case 2:
		log.Printf("[TrustTunnel Core] INFO: %s", goMsg)
	case 3, 4:
		log.Printf("[TrustTunnel Core] DEBUG: %s", goMsg)
	default:
		log.Printf("[TrustTunnel Core] %s", goMsg)
	}
}

//export go_protect_socket
func go_protect_socket(fd C.int) C.int {
	// For mobile platforms, use the global callback if set
	if globalProtectCallback != nil {
		// Call the Kotlin/Swift function
		// Mobile will return 1 for True (Protected), 0 for False
		res := C.execute_protect_cb(globalProtectCallback, fd)
		if res == 1 {
			return 0 // TrustTunnel C++ expects 0 for Success
		}
		return -1 // TrustTunnel C++ expects -1 for Failure
	}
	// For desktop platforms, default to allow
	return 0
}

type TrustTunnelManager struct {
	tomlConfig string
}

func NewTrustTunnelManager(tomlConfig string) *TrustTunnelManager {
	return &TrustTunnelManager{
		tomlConfig: tomlConfig,
	}
}

// Start launches the VPN with the given TOML configuration.
func (m *TrustTunnelManager) Start() error {
	// Hook the logger BEFORE starting the engine
	C.dobby_vpn_set_log_callback((C.dobby_on_log_message_t)(C.c_log_message))

	// For mobile platforms, set the protect callback
	C.dobby_vpn_set_protect_callback((C.dobby_on_protect_socket_t)(C.c_protect_cb))

	cConfig := C.CString(m.tomlConfig)
	defer C.free(unsafe.Pointer(cConfig))

	// Pass the TOML string, the bridged C callback, and a nil pointer for the argument
	C.dobby_vpn_start(cConfig, (C.dobby_on_state_changed_t)(C.c_state_changed), nil)
	return nil
}

// Stop halts the VPN and frees resources.
func (m *TrustTunnelManager) Stop() {
	C.dobby_vpn_stop()
}
