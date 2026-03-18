//go:build !(android || ios)
// +build !android,!ios

package main

import (
	"fmt"
	trusttunnel "go_client/trusttunnel"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	// 1. Define the TOML configuration.
	// Replace the [endpoint] details with your actual server credentials.
	tomlConfig := `
loglevel = "info"
vpn_mode = "general"
killswitch_enabled = false

[endpoint]
hostname = "vpn.example.com"
addresses = ["198.51.100.1:443"]
has_ipv6 = true
username = "myuser"
password = "mypassword"
client_random = ""
skip_verification = true
upstream_protocol = "http2"

[listener.socks]
address = "127.0.0.1:1080"
username = ""
password = ""
`

	// 2. Initialize and start the manager
	fmt.Println("Starting the application")
	manager := trusttunnel.NewTrustTunnelManager()
	fmt.Println("Starting TrustTunnel SOCKS5 Proxy on 127.0.0.1:1080...")

	if err := manager.Start(tomlConfig); err != nil {
		fmt.Printf("Failed to start TrustTunnel: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Proxy is running. Press Ctrl+C to stop.")

	// 3. Block and wait for an interrupt signal (Ctrl+C)
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c

	// 4. Gracefully shut down
	fmt.Println("\nStopping TrustTunnel...")
	manager.Stop()
	fmt.Println("Stopped.")
}
