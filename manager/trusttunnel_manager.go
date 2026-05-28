package manager

/*
#cgo CFLAGS: -I${SRCDIR}

// --- Static Linking Configuration ---
//
// Link against bundled static libraries (.a/.lib) which include all dependencies.
// This approach provides better portability and eliminates runtime dependency issues.
//
// CRITICAL: The static libraries are built with specific C++ toolchains.
// CGO must use the SAME C++ standard library to avoid ABI incompatibility crashes.

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
