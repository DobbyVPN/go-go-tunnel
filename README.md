# TrustTunnel-Go

A Go wrapper for the TrustTunnel VPN library, providing cross-platform support for Windows, Linux, macOS, iOS, and Android.

## Overview

This project provides a Go wrapper around the TrustTunnel VPN protocol engine. It uses CGO to call platform-specific C bridge implementations that interface with the TrustTunnel core library.

## Architecture

```
Go Application
    ↓ (CGO)
Platform-Specific C Bridge (dobby_bridge_*.cpp/mm)
    ↓
TrustTunnel Core Library (platform-agnostic)
```

### Components

- **[`dobby_bridge_common.h`](dobby_bridge/dobby_bridge_common.h)** - Common header with platform-agnostic C API declarations
- **[`dobby_bridge_windows.cpp`](dobby_bridge/dobby_bridge_windows.cpp)** - Windows-specific implementation using Wintun
- **[`dobby_bridge_unix.cpp`](dobby_bridge/dobby_bridge_unix.cpp)** - Unix-specific implementation for Linux and macOS
- **[`dobby_bridge_android.cpp`](dobby_bridge/dobby_bridge_android.cpp)** - Android-specific implementation using JNI
- **[`dobby_bridge_ios.mm`](dobby_bridge/dobby_bridge_ios.mm)** - iOS-specific implementation using Objective-C++
- **[`trusttunnel_manager.go`](trusttunnel_manager.go)** - Unified Go wrapper using CGO for all platforms
- **[`CMakeLists.txt`](dobby_bridge/CMakeLists.txt)** - Build configuration with conditional compilation

## Features

- ✅ Cross-platform support (Windows, Linux, macOS, iOS, Android)
- ✅ Platform-specific socket protection
- ✅ Customizable socket protection callback
- ✅ State change notifications
- ✅ Logging callbacks
- ✅ TOML configuration support

## Building

### Prerequisites

- Go 1.21 or higher
- CMake 3.24 or higher
- C++20 compatible compiler
- TrustTunnel submodule (included)

### Build Steps

1. **Build the C library:**

```bash
cd TrustTunnelClient
mkdir build && cd build
cmake -DCMAKE_BUILD_TYPE=Release ..
cmake --build . --target vpnlibs_core vpnlibs_trusttunnel
```

2. **Build the platform-specific bridge:**

```bash
cd ../dobby_bridge
mkdir build && cd build
cmake -DCMAKE_BUILD_TYPE=Release ..
cmake --build .
```

3. **Build the Go package:**

```bash
cd ../..
go build ./dobby_bridge
```

### Platform-Specific Build Notes

#### Windows
```bash
cmake -G "Ninja" -DCMAKE_C_COMPILER=cl.exe -DCMAKE_CXX_COMPILER=cl.exe ..
```

#### macOS
```bash
cmake -DCMAKE_C_COMPILER=clang -DCMAKE_CXX_COMPILER=clang++ -DCMAKE_CXX_FLAGS="-stdlib=libc++" ..
```

#### Linux
```bash
cmake -DCMAKE_C_COMPILER=clang -DCMAKE_CXX_COMPILER=clang++ -DCMAKE_CXX_FLAGS="-stdlib=libc++" ..
```

#### Android
```bash
# Use Android NDK toolchain
cmake -DCMAKE_TOOLCHAIN_FILE=$NDK/build/cmake/android.toolchain.cmake -DANDROID_ABI=arm64-v8a ..
```

#### iOS
```bash
# Use iOS toolchain
cmake -DCMAKE_SYSTEM_NAME=iOS -DCMAKE_OSX_ARCHITECTURES=arm64 ..
```

## Usage

### Basic Example

```go
package main

import (
    "log"
    "time"
    
    "github.com/yourusername/trusttunnel-go/dobby_bridge"
)

func main() {
    bridge := dobby_bridge.NewDobbyBridge()
    
    // Set up callbacks
    bridge.SetStateChangedCallback(func(state dobby_bridge.VpnState) {
        log.Printf("VPN State changed: %v", state)
    })
    
    bridge.SetLogCallback(func(level dobby_bridge.LogLevel, message string) {
        log.Printf("[%v] %s", level, message)
    })
    
    // Start VPN
    config := `
[server]
address = "your-server.com"
port = 443

[credentials]
username = "your-username"
password = "your-password"

[listener]
type = "socks"
address = "127.0.0.1:1080"
`
    
    if err := bridge.Start(config); err != nil {
        log.Fatal(err)
    }
    
    time.Sleep(30 * time.Second)
    
    if err := bridge.Stop(); err != nil {
        log.Fatal(err)
    }
}
```

### Custom Socket Protection

You can provide your own socket protection implementation:

```go
bridge.SetProtectSocketCallback(func(fd int) int {
    // Your custom socket protection logic
    // Return 0 on success, non-zero on failure
    return 0
})
```

#### Platform-Specific Socket Protection Examples

**Android:**
```go
bridge.SetProtectSocketCallback(func(fd int) int {
    // Call VpnService.protect(fd) via JNI
    return protectSocketViaJNI(fd)
})
```

**iOS:**
```go
bridge.SetProtectSocketCallback(func(fd int) int {
    // Bind socket to specific interface
    return bindSocketToInterface(fd, "en0")
})
```

**Linux:**
```go
bridge.SetProtectSocketCallback(func(fd int) int {
    // Use SO_BINDTODEVICE
    return bindSocketToDevice(fd, "eth0")
})
```

**Windows:**
```go
bridge.SetProtectSocketCallback(func(fd int) int {
    // Use Wintun API
    return protectSocketViaWintun(fd)
})
```

## API Reference

### Types

#### `VpnState`
```go
type VpnState int

const (
    VpnStateDisconnected VpnState = iota
    VpnStateConnecting
    VpnStateConnected
    VpnStateWaitingRecovery
    VpnStateRecovering
    VpnStateWaitingForNetwork
)
```

#### `LogLevel`
```go
type LogLevel int

const (
    LogLevelTrace LogLevel = iota
    LogLevelDebug
    LogLevelInfo
    LogLevelWarn
    LogLevelError
)
```

### Callbacks

#### `StateChangedCallback`
```go
type StateChangedCallback func(state VpnState)
```

Called when the VPN state changes.

#### `LogCallback`
```go
type LogCallback func(level LogLevel, message string)
```

Called when the VPN library logs a message.

#### `ProtectSocketCallback`
```go
type ProtectSocketCallback func(fd int) int
```

Called to protect a socket from being routed through the VPN. Return 0 on success, non-zero on failure.

### Methods

#### `NewDobbyBridge() *DobbyBridge`
Creates a new DobbyBridge instance.

#### `SetStateChangedCallback(cb StateChangedCallback)`
Sets the callback that will be called when the VPN state changes.

#### `SetLogCallback(cb LogCallback)`
Sets the callback that will be called when the VPN library logs a message.

#### `SetProtectSocketCallback(cb ProtectSocketCallback)`
Sets the callback that will be called to protect a socket.

#### `Start(config string) error`
Starts the VPN connection with the given TOML configuration.

#### `Stop() error`
Stops the VPN connection.

## Configuration

The VPN is configured using TOML format. See the TrustTunnel documentation for full configuration options.

### Minimal Configuration

```toml
[server]
address = "your-server.com"
port = 443

[credentials]
username = "your-username"
password = "your-password"

[listener]
type = "socks"
address = "127.0.0.1:1080"

[upstream]
protocol = "http2"
location = "us-east"
```

## Platform-Specific Notes

### Windows
- Uses Wintun for TUN device
- Requires Windows 7 or later
- Socket protection uses `vpn_win_socket_protect()`

### Linux
- Uses TUN/TAP device
- Socket protection can use `SO_BINDTODEVICE` or network namespaces
- Requires root privileges for TUN device

### macOS
- Uses utun device
- Socket protection uses `IP_BOUND_IF`/`IPV6_BOUND_IF` socket options
- Requires root privileges for TUN device

### iOS
- Uses NetworkExtension framework
- Socket protection uses `IP_BOUND_IF`/`IPV6_BOUND_IF` socket options
- Requires proper entitlements

### Android
- Uses VpnService API
- Socket protection uses `VpnService.protect()`
- Requires proper permissions

## Troubleshooting

### Build Errors

**Error: `aclapi.h` file not found**
- This error occurs when trying to build Windows-specific code on non-Windows platforms
- Solution: Ensure you're using the correct platform-specific source file

**Error: `windows.h` file not found**
- This error occurs when trying to build Windows-specific code on non-Windows platforms
- Solution: Ensure you're using the correct platform-specific source file

### Runtime Errors

**VPN fails to start**
- Check the TOML configuration
- Verify server credentials
- Check network connectivity
- Review logs via the log callback

**Socket protection fails**
- Ensure your socket protection callback is implemented correctly
- Check platform-specific requirements (e.g., root privileges on Linux)
- Verify the socket file descriptor is valid

## Contributing

Contributions are welcome! Please ensure:

1. Code follows Go best practices
2. Platform-specific code is properly isolated
3. Changes are tested on all supported platforms
4. Documentation is updated

## License

This project uses the TrustTunnel library. Please refer to the TrustTunnel license for the underlying library.

## Acknowledgments

- [TrustTunnel](https://github.com/AdguardTeam/TrustTunnel) - The underlying VPN protocol engine
- [AdGuard](https://adguard.com/) - The creators of TrustTunnel
