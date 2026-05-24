package manager

/*
#cgo CFLAGS: -I${SRCDIR}

// --- Bundled Static Linking Configuration ---
//
// The libdobby_bridge.a library is a "fat" static library that includes:
// - dobby_bridge code (our C++ wrapper)
// - TrustTunnel core libraries (vpnlibs_trusttunnel, vpnlibs_core, vpnlibs_net, vpnlibs_common)
// - OpenSSL (libssl, libcrypto)
// - libevent (event, event_core, event_extra, event_openssl, event_pthreads)
// - Other dependencies (fmt, magic_enum, tomlplusplus, etc.)
//
// We only need to link system libraries and frameworks that aren't included.

// macOS: Link bundled static library with required system frameworks
#cgo darwin LDFLAGS: -L${SRCDIR}/../lib/macos -ldobby_bridge -framework CoreFoundation -framework Security -framework Network -framework SystemConfiguration -lc++ -lresolv -lz

// Linux: Link bundled static library with required system libraries
#cgo linux LDFLAGS: -L${SRCDIR}/../lib/linux -ldobby_bridge -lpthread -ldl -lc++ -lc++abi -lm -lresolv -lz

// Windows: Link bundled static library with MSVC runtime and system libraries
// Note: The static library is built with MSVC, so we need to link against the MSVC runtime
// Windows: Link bundled static library with MSVC runtime and system libraries
#cgo windows,amd64 LDFLAGS: -L${SRCDIR}/../lib/windows -ldobby_bridge -lws2_32 -liphlpapi -lcrypt32 -lbcrypt -ladvapi32 -lkernel32 -luser32 -lucrt -lvcruntime -lmsvcrt -Wl,--allow-multiple-definition
#cgo windows,386 LDFLAGS: -L${SRCDIR}/../lib/windows -ldobby_bridge -lws2_32 -liphlpapi -lcrypt32 -lbcrypt -ladvapi32 -lkernel32 -luser32 -lucrt -lvcruntime -lmsvcrt -Wl,--allow-multiple-definition

// iOS: Link static library explicitly (Apple strongly prefers static linking)
#cgo ios,arm64 LDFLAGS: ${SRCDIR}/../lib/ios/libdobby_bridge.a -framework Foundation -framework NetworkExtension -framework Network -lc++ -lresolv

// Android: Link static library for each ABI (architecture)
// ARM64 (most common - modern 64-bit ARM devices)
#cgo android,arm64 LDFLAGS: ${SRCDIR}/../lib/android/arm64-v8a/libdobby_bridge.a -llog -lm -lc++_static

// ARM32 (legacy 32-bit ARM devices)
#cgo android,arm LDFLAGS: ${SRCDIR}/../lib/android/armeabi-v7a/libdobby_bridge.a -llog -lm -lc++_static

// x86_64 (64-bit x86 - emulators and some tablets)
#cgo android,amd64 LDFLAGS: ${SRCDIR}/../lib/android/x86_64/libdobby_bridge.a -llog -lm -lc++_static

// x86 (32-bit x86 - old emulators)
#cgo android,386 LDFLAGS: ${SRCDIR}/../lib/android/x86/libdobby_bridge.a -llog -lm -lc++_static

#include <stdlib.h>
#include "../dobby_bridge/dobby_bridge_common.h"

// C bridge functions (defined in bridge.c)
extern void c_state_changed(void* arg, int state);
extern void c_log_message(int level, const char* msg);
extern int c_protect_cb(int fd);
*/
import "C"
import (
	"sync"
	"unsafe"
)

// VpnState represents the VPN connection state
type VpnState int

const (
	StateDisconnected VpnState = iota
	StateConnecting
	StateConnected
	StateReconnecting
	StateError
)

// LogLevel represents the log level
type LogLevel int

const (
	LogError LogLevel = iota
	LogWarn
	LogInfo
	LogDebug
	LogTrace
)

// ProtectSocketFunc is the type for socket protection callbacks
// Returns 0 on success, non-zero on failure
type ProtectSocketFunc func(fd int) int

// StateChangedFunc is the type for state change callbacks
type StateChangedFunc func(state VpnState)

// LogFunc is the type for log callbacks
type LogFunc func(level LogLevel, message string)

// TrustTunnelManager is the main wrapper for the TrustTunnel VPN library
type TrustTunnelManager struct {
	config          string
	protectCallback ProtectSocketFunc
	stateCallback   StateChangedFunc
	logCallback     LogFunc
	mu              sync.RWMutex
}

// NewTrustTunnelManager creates a new TrustTunnelManager instance
func NewTrustTunnelManager() *TrustTunnelManager {
	return &TrustTunnelManager{}
}

// SetProtectSocketCallback sets a custom socket protection callback
// This is useful for mobile platforms where you need to protect sockets
// from the VPN tunnel (e.g., Android's VpnService.protect() or iOS NEPacketTunnelFlow)
func (t *TrustTunnelManager) SetProtectSocketCallback(cb ProtectSocketFunc) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.protectCallback = cb
}

// SetStateChangedCallback sets a callback for VPN state changes
func (t *TrustTunnelManager) SetStateChangedCallback(cb StateChangedFunc) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.stateCallback = cb
}

// SetLogCallback sets a callback for log messages from the VPN core
func (t *TrustTunnelManager) SetLogCallback(cb LogFunc) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.logCallback = cb
}

// Start launches the VPN with the given TOML configuration
func (t *TrustTunnelManager) Start(tomlConfig string) error {
	t.mu.Lock()
	t.config = tomlConfig
	t.mu.Unlock()

	SetGlobalManager(t)

	C.dobby_vpn_set_log_callback((C.dobby_on_log_message_t)(C.c_log_message))

	C.dobby_vpn_set_protect_callback((C.dobby_on_protect_socket_t)(C.c_protect_cb))

	cConfig := C.CString(tomlConfig)
	defer C.free(unsafe.Pointer(cConfig))

	// Pass the TOML string, the bridged C callback, and a nil pointer for the argument
	C.dobby_vpn_start(cConfig, (C.dobby_on_state_changed_t)(C.c_state_changed), nil)
	return nil
}

// Stop halts the VPN and frees resources
func (t *TrustTunnelManager) Stop() {
	C.dobby_vpn_stop()
}

// Global instance for C callbacks
var globalManager *TrustTunnelManager

// SetGlobalManager sets the global manager instance for C callbacks
// This is called automatically by Start()
func SetGlobalManager(manager *TrustTunnelManager) {
	globalManager = manager
}

//export go_state_changed
func go_state_changed(arg unsafe.Pointer, state C.int) {
	if globalManager == nil {
		return
	}
	globalManager.mu.RLock()
	cb := globalManager.stateCallback
	globalManager.mu.RUnlock()

	if cb != nil {
		cb(VpnState(state))
	}
}

//export go_log_message
func go_log_message(level C.int, msg *C.char) {
	if globalManager == nil {
		return
	}
	globalManager.mu.RLock()
	cb := globalManager.logCallback
	globalManager.mu.RUnlock()

	if cb != nil {
		goMsg := C.GoString(msg)
		cb(LogLevel(level), goMsg)
	}
}

//export go_protect_socket
func go_protect_socket(fd C.int) C.int {
	if globalManager == nil {
		return 0 // Default to allow
	}
	globalManager.mu.RLock()
	cb := globalManager.protectCallback
	globalManager.mu.RUnlock()

	if cb != nil {
		return C.int(cb(int(fd)))
	}
	return 0 // Default to allow
}
