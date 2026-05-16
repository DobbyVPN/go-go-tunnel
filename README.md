# TrustTunnel-Go

A Go wrapper for the TrustTunnel VPN library, providing cross-platform support for Windows, Linux, macOS, iOS, and Android.

[![Build Desktop](https://github.com/your-repo/TrustTunnel-Go/actions/workflows/build-desktop.yml/badge.svg)](https://github.com/your-repo/TrustTunnel-Go/actions/workflows/build-desktop.yml)
[![Build Mobile](https://github.com/your-repo/TrustTunnel-Go/actions/workflows/build-mobile.yml/badge.svg)](https://github.com/your-repo/TrustTunnel-Go/actions/workflows/build-mobile.yml)

## Overview

TrustTunnel-Go provides a pure Go API wrapper around the TrustTunnel VPN protocol engine. It uses CGO to interface with platform-specific C++ implementations, allowing Go applications to leverage TrustTunnel's powerful VPN capabilities.

### Key Features

- ✅ **Pure Go API** - No need to import `"C"` in your application code
- ✅ **Cross-platform** - Windows, Linux, macOS, iOS, Android
- ✅ **Type-safe callbacks** - State changes, logging, and socket protection
- ✅ **Thread-safe** - All operations protected with mutexes
- ✅ **TOML configuration** - Simple, readable configuration format

## Architecture

```
┌─────────────────────────────────────┐
│      Your Go Application            │
│  (Pure Go - No CGO Required)        │
└──────────────┬──────────────────────┘
               │ Import
               ↓
┌─────────────────────────────────────┐
│   manager/trusttunnel_manager.go    │
│        (CGO Wrapper Layer)          │
└──────────────┬──────────────────────┘
               │ CGO
               ↓
┌─────────────────────────────────────┐
│     manager/bridge.c                │
│  (C Callback Bridge Functions)      │
└──────────────┬──────────────────────┘
               │ Link
               ↓
┌─────────────────────────────────────┐
│   libdobby_bridge.{dylib,so,dll}    │
│  (Platform-Specific C++ Library)    │
└──────────────┬──────────────────────┘
               │
               ↓
┌─────────────────────────────────────┐
│    TrustTunnel C++ Core Engine      │
│  (VPN Protocol Implementation)      │
└─────────────────────────────────────┘
```

### Components

- **[`manager/trusttunnel_manager.go`](manager/trusttunnel_manager.go)** - Pure Go API with CGO bindings
- **[`manager/bridge.c`](manager/bridge.c)** - C callback bridge to Go functions
- **[`dobby_bridge/`](dobby_bridge/)** - Platform-specific C++ implementations
  - [`dobby_bridge_common.h`](dobby_bridge/dobby_bridge_common.h) - Common C API
  - [`dobby_bridge_windows.cpp`](dobby_bridge/dobby_bridge_windows.cpp) - Windows (Wintun)
  - [`dobby_bridge_unix.cpp`](dobby_bridge/dobby_bridge_unix.cpp) - Linux/macOS (TUN/TAP)
  - [`dobby_bridge_android.cpp`](dobby_bridge/dobby_bridge_android.cpp) - Android (JNI)
  - [`dobby_bridge_ios.mm`](dobby_bridge/dobby_bridge_ios.mm) - iOS (NetworkExtension)
- **[`examples/example.go`](examples/example.go)** - Example usage

## Quick Start

### Installation

1. **Get the package:**
   ```bash
   go get github.com/your-repo/TrustTunnel-Go/manager
   ```

2. **Download platform-specific library:**

   Download the pre-built library for your platform from [GitHub Releases](https://github.com/your-repo/TrustTunnel-Go/releases):

   - **macOS**: `libdobby_bridge.dylib`
   - **Linux**: `libdobby_bridge.so`
   - **Windows**: `dobby_bridge.dll` and `dobby_bridge.lib`
   - **Android**: `libdobby_bridge.so` (from android/arm64-v8a)
   - **iOS**: `libdobby_bridge.a`

3. **Set up library directory structure:**
   ```bash
   mkdir -p lib/macos    # or lib/linux, lib/windows, etc.
   mv libdobby_bridge.dylib lib/macos/
   ```

   Or use the provided setup script:
   ```bash
   ./setup_libs.sh
   ```

4. **Build your application:**
   ```bash
   go build
   ```

### Basic Example

```go
package main

import (
    "log"
    "time"
    
    tt "github.com/your-repo/TrustTunnel-Go/manager"
)

func main() {
    // Create manager
    manager := tt.NewTrustTunnelManager()
    
    // Set up state change callback
    manager.SetStateChangedCallback(func(state tt.VpnState) {
        switch state {
        case tt.StateConnected:
            log.Println("✅ VPN Connected!")
        case tt.StateDisconnected:
            log.Println("❌ VPN Disconnected")
        case tt.StateError:
            log.Println("⚠️ VPN Error")
        }
    })
    
    // Set up logging callback
    manager.SetLogCallback(func(level tt.LogLevel, message string) {
        log.Printf("[%v] %s", level, message)
    })
    
    // Optional: Custom socket protection
    manager.SetProtectSocketCallback(func(fd int) int {
        log.Printf("Protecting socket fd: %d", fd)
        return 0 // Return 0 on success
    })
    
    // Configure VPN
    config := `
[server]
address = "vpn.example.com"
port = 443

[credentials]
username = "user"
password = "pass"

[listener]
type = "socks"
address = "127.0.0.1:1080"

[upstream]
protocol = "http2"
location = "us-east"
`
    
    // Start VPN
    if err := manager.Start(config); err != nil {
        log.Fatal(err)
    }
    
    // Keep running
    log.Println("VPN is running. Press Ctrl+C to stop.")
    time.Sleep(30 * time.Second)
    
    // Stop VPN
    manager.Stop()
    log.Println("VPN stopped.")
}
```

## Building from Source

### Prerequisites

- **Go** 1.21 or higher
- **CMake** 3.24 or higher
- **C++20 compatible compiler**
  - macOS: Clang (from Xcode Command Line Tools)
  - Linux: Clang 21+ or GCC 13+
  - Windows: MSVC 2022 or MinGW
- **Python** 3.8+ (for Conan dependency management)
- **Ninja** build system

### Build Steps

#### 1. Clone Repository

```bash
git clone --recursive https://github.com/your-repo/TrustTunnel-Go.git
cd TrustTunnel-Go
```

The `--recursive` flag is important to fetch the TrustTunnel submodule.

#### 2. Build Platform Library

**macOS:**
```bash
cd TrustTunnelClient
python3 ./scripts/bootstrap_conan_deps.py
echo 'add_subdirectory("../dobby_bridge" "dobby_bridge")' >> CMakeLists.txt

mkdir build && cd build
cmake -DCMAKE_BUILD_TYPE=Release \
      -DCMAKE_C_COMPILER=clang \
      -DCMAKE_CXX_COMPILER=clang++ \
      -DCMAKE_CXX_FLAGS="-stdlib=libc++" \
      -G "Ninja" ..
cmake --build . --target dobby_bridge

# Copy library
cp dobby_bridge/libdobby_bridge.dylib ../../lib/macos/
```

**Linux:**
```bash
cd TrustTunnelClient
python3 ./scripts/bootstrap_conan_deps.py
echo 'add_subdirectory("../dobby_bridge" "dobby_bridge")' >> CMakeLists.txt

mkdir build && cd build
cmake -DCMAKE_BUILD_TYPE=Release \
      -DCMAKE_C_COMPILER=clang \
      -DCMAKE_CXX_COMPILER=clang++ \
      -DCMAKE_CXX_FLAGS="-stdlib=libc++" \
      -G "Ninja" ..
cmake --build . --target dobby_bridge

# Copy library
cp dobby_bridge/libdobby_bridge.so ../../lib/linux/
```

**Windows (PowerShell):**
```powershell
cd TrustTunnelClient
python ./scripts/bootstrap_conan_deps.py
Add-Content -Path CMakeLists.txt -Value 'add_subdirectory("../dobby_bridge" "dobby_bridge")'

mkdir build; cd build
cmake -DCMAKE_BUILD_TYPE=Release `
      -DCMAKE_C_COMPILER=cl.exe `
      -DCMAKE_CXX_COMPILER=cl.exe `
      -G "Ninja" ..
cmake --build . --target dobby_bridge

# Copy libraries
Copy-Item dobby_bridge\*.dll ..\..\lib\windows\
Copy-Item dobby_bridge\*.lib ..\..\lib\windows\
```

#### 3. Build Go Package

```bash
cd ../..  # Back to project root
go build ./manager
```

#### 4. Test with Example

```bash
cd examples
go build -v
./examples  # or .\examples.exe on Windows
```

## API Reference

### Types

#### VpnState
```go
type VpnState int

const (
    StateDisconnected VpnState = iota  // VPN is disconnected
    StateConnecting                     // VPN is connecting
    StateConnected                      // VPN is connected
    StateReconnecting                   // VPN is reconnecting
    StateError                          // VPN encountered an error
)
```

#### LogLevel
```go
type LogLevel int

const (
    LogError LogLevel = iota  // Error messages
    LogWarn                    // Warning messages
    LogInfo                    // Informational messages
    LogDebug                   // Debug messages
    LogTrace                   // Trace messages (very verbose)
)
```

### Callback Types

#### ProtectSocketFunc
```go
type ProtectSocketFunc func(fd int) int
```
Called when a socket needs to be protected from the VPN tunnel. Return `0` on success, non-zero on failure.

**Platform-specific implementations:**
- **Android**: Call `VpnService.protect(fd)` via JNI
- **iOS**: Use `IP_BOUND_IF`/`IPV6_BOUND_IF` socket options
- **Linux**: Use `SO_BINDTODEVICE` or network namespaces
- **Windows**: Use Wintun API
- **macOS**: Use `IP_BOUND_IF`/`IPV6_BOUND_IF` socket options

#### StateChangedFunc
```go
type StateChangedFunc func(state VpnState)
```
Called when the VPN connection state changes.

#### LogFunc
```go
type LogFunc func(level LogLevel, message string)
```
Called when the VPN library emits a log message.

### TrustTunnelManager Methods

#### NewTrustTunnelManager
```go
func NewTrustTunnelManager() *TrustTunnelManager
```
Creates a new TrustTunnelManager instance.

#### SetStateChangedCallback
```go
func (t *TrustTunnelManager) SetStateChangedCallback(cb StateChangedFunc)
```
Sets the callback for VPN state changes. Thread-safe.

#### SetLogCallback
```go
func (t *TrustTunnelManager) SetLogCallback(cb LogFunc)
```
Sets the callback for log messages. Thread-safe.

#### SetProtectSocketCallback
```go
func (t *TrustTunnelManager) SetProtectSocketCallback(cb ProtectSocketFunc)
```
Sets the callback for socket protection. Thread-safe.

**Example:**
```go
manager.SetProtectSocketCallback(func(fd int) int {
    // Your platform-specific protection logic
    return 0
})
```

#### Start
```go
func (t *TrustTunnelManager) Start(tomlConfig string) error
```
Starts the VPN with the given TOML configuration.

#### Stop
```go
func (t *TrustTunnelManager) Stop()
```
Stops the VPN and frees resources.

## Configuration

TrustTunnel uses TOML format for configuration.

### Minimal Configuration

```toml
[server]
address = "vpn.example.com"
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

### Advanced Configuration

```toml
[server]
address = "vpn.example.com"
port = 443
timeout = 30

[credentials]
username = "your-username"
password = "your-password"

[listener]
type = "tun"  # or "socks"
address = "10.0.0.1"
mtu = 1500

[upstream]
protocol = "http3"  # or "http2"
location = "us-east"
fallback_location = "us-west"

[dns]
servers = ["8.8.8.8", "8.8.4.4"]
bootstrap = ["1.1.1.1"]

[routing]
mode = "split"  # or "full"
exclude_ips = ["192.168.0.0/16"]
```

See the TrustTunnel documentation for complete configuration options.

## Platform-Specific Notes

### macOS
- **TUN Device**: Uses `utun` device
- **Permissions**: Requires root privileges for TUN device creation
- **Socket Protection**: Uses `IP_BOUND_IF`/`IPV6_BOUND_IF` socket options
- **Frameworks**: Requires CoreFoundation and Security frameworks

### Linux
- **TUN Device**: Uses TUN/TAP device (`/dev/net/tun`)
- **Permissions**: Requires `CAP_NET_ADMIN` capability or root privileges
- **Socket Protection**: Can use `SO_BINDTODEVICE` or network namespaces
- **Dependencies**: Requires `libc++`, `pthread`, `dl`

### Windows
- **TUN Device**: Uses Wintun driver
- **Permissions**: Requires administrator privileges
- **Socket Protection**: Uses Wintun API
- **Build**: Requires Visual Studio 2022 or MinGW

### iOS
- **Network Extension**: Uses NetworkExtension framework
- **Entitlements**: Requires proper entitlements (`com.apple.developer.networking.networkextension`)
- **Socket Protection**: Uses `IP_BOUND_IF`/`IPV6_BOUND_IF`
- **Deployment**: Requires provisioning profile with Network Extension capability

### Android
- **VPN Service**: Uses VpnService API
- **Permissions**: Requires `BIND_VPN_SERVICE` permission
- **Socket Protection**: Uses `VpnService.protect(fd)`
- **NDK**: Requires Android NDK for building

## Troubleshooting

### Build Issues

#### Error: "library 'dobby_bridge' not found"

**Cause**: The platform-specific library is not in the expected location.

**Solution**:
```bash
# Check if library exists
ls -la lib/macos/libdobby_bridge.dylib  # macOS
ls -la lib/linux/libdobby_bridge.so      # Linux
ls -la lib/windows/dobby_bridge.dll      # Windows

# If missing, run setup script
./setup_libs.sh

# Or download from GitHub Releases
curl -L -o lib/macos/libdobby_bridge.dylib <release-url>
```

#### Error: "search path not found"

**Cause**: The `lib/` directory structure doesn't exist.

**Solution**:
```bash
mkdir -p lib/macos lib/linux lib/windows lib/android/arm64-v8a lib/ios
./setup_libs.sh
```

#### Error: "conflicting types for 'go_log_message'"

**Cause**: Missing or incorrect [`manager/bridge.c`](manager/bridge.c) file.

**Solution**: Ensure `manager/bridge.c` exists with the correct bridge functions.

### Runtime Issues

#### macOS: "dyld: Library not loaded"

**Cause**: Library not found at runtime.

**Solution**:
```bash
# Option 1: Copy library next to executable
cp lib/macos/libdobby_bridge.dylib .

# Option 2: Set DYLD_LIBRARY_PATH
export DYLD_LIBRARY_PATH=$PWD/lib/macos:$DYLD_LIBRARY_PATH

# Option 3: Use install_name_tool to fix rpath
install_name_tool -id @rpath/libdobby_bridge.dylib lib/macos/libdobby_bridge.dylib
install_name_tool -add_rpath @executable_path/lib/macos ./your-app
```

#### Linux: "error while loading shared libraries"

**Cause**: Library not in library search path.

**Solution**:
```bash
# Option 1: Set LD_LIBRARY_PATH
export LD_LIBRARY_PATH=$PWD/lib/linux:$LD_LIBRARY_PATH

# Option 2: Copy to system library directory
sudo cp lib/linux/libdobby_bridge.so /usr/local/lib/
sudo ldconfig

# Option 3: Use rpath during build
go build -ldflags="-r /path/to/lib/linux"
```

#### Windows: "The code execution cannot proceed because dobby_bridge.dll was not found"

**Cause**: DLL not in executable directory or system PATH.

**Solution**:
```powershell
# Copy DLL next to executable
Copy-Item lib\windows\dobby_bridge.dll .

# Or add to PATH
$env:PATH += ";$PWD\lib\windows"
```

### VPN Connection Issues

#### VPN fails to connect

1. **Check configuration**: Verify TOML syntax and server details
2. **Check logs**: Set up log callback to see detailed error messages
3. **Network connectivity**: Ensure server is reachable
4. **Firewall**: Check if firewall blocks VPN traffic
5. **Permissions**: Ensure app has necessary permissions (root/admin)

#### Socket protection fails

1. **Platform-specific**: Ensure correct protection implementation for your platform
2. **Permissions**: Verify app has necessary permissions
3. **File descriptor**: Check if fd is valid before protecting

## Why This Approach?

### The CGO Challenge

Creating a "go gettable" library with CGO and shared libraries is challenging because:

1. **CGO requires libraries at compile time**: The `-L` flag specifies where to find libraries during linking
2. **Shared libraries can't be in go.mod**: Binary files shouldn't be in git repositories
3. **Platform-specific binaries**: Different OS/architectures need different libraries

### Our Solution

We provide:
- ✅ **Pure Go API**: Your code doesn't import `"C"`
- ✅ **GitHub Actions**: Automated builds for all platforms
- ✅ **GitHub Releases**: Pre-built libraries available for download
- ✅ **Setup script**: Easy library organization
- ✅ **Clear documentation**: Step-by-step instructions

### Alternative Approaches

If you need truly "go gettable" without manual library placement:

1. **Static Linking**: Link static libraries (`.a`) and include them in git
   - Pros: Single `go get`, no runtime dependencies
   - Cons: Larger repo size, larger binaries

2. **Vendoring**: Include pre-built libraries in vendor directory
   - Pros: Everything in one repo
   - Cons: Large repo, version control bloat

3. **Pure Go Rewrite**: Rewrite TrustTunnel in pure Go
   - Pros: Truly go gettable
   - Cons: Massive effort, feature parity challenges

We chose the hybrid approach for the best balance of usability and practicality.

## Contributing

Contributions are welcome! Please:

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Make your changes
4. Test on all supported platforms (use GitHub Actions)
5. Commit with clear messages (`git commit -m 'Add amazing feature'`)
6. Push to your branch (`git push origin feature/amazing-feature`)
7. Open a Pull Request

### Development Guidelines

- Follow Go best practices and `golint` recommendations
- Keep platform-specific code isolated
- Update documentation for any API changes
- Add tests for new functionality
- Ensure backward compatibility

## License

This project wraps the TrustTunnel library. Please refer to the [TrustTunnel license](TrustTunnelClient/LICENSE) for the underlying library terms.

## Acknowledgments

- [TrustTunnel](https://github.com/AdguardTeam/TrustTunnel) - The powerful VPN protocol engine
- [AdGuard](https://adguard.com/) - Creators of TrustTunnel
- All contributors to this project

## Support

- **Issues**: [GitHub Issues](https://github.com/your-repo/TrustTunnel-Go/issues)
- **Discussions**: [GitHub Discussions](https://github.com/your-repo/TrustTunnel-Go/discussions)
- **Documentation**: [Wiki](https://github.com/your-repo/TrustTunnel-Go/wiki)

---

Made with ❤️ for the Go and VPN communities
