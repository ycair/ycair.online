//go:build windows

package tun

import (
	"fmt"
	"os/exec"
	"strings"

	"golang.zx2c4.com/wireguard/tun"
)

// Interface is a TUN virtual network interface backed by the Wintun driver
// via wireguard-go. On Windows, it creates a Wintun-based Layer 3 TUN adapter
// configured via netsh.
type Interface struct {
	dev  tun.Device
	name string
	ip   string
}

// Create creates a new Wintun-based TUN interface with the given IP address.
// Requires administrator privileges on first run (driver installation).
// The Wintun driver DLL is embedded by wireguard-go — no manual install needed.
func Create(ip string) (*Interface, error) {
	dev, err := tun.CreateTUN("ycair", MTU)
	if err != nil {
		return nil, fmt.Errorf("create wintun tun: %w (run as administrator?)", err)
	}

	name, err := dev.Name()
	if err != nil {
		dev.Close()
		return nil, fmt.Errorf("get tun name: %w", err)
	}

	t := &Interface{
		dev:  dev,
		name: name,
		ip:   ip,
	}

	if err := t.configure(); err != nil {
		dev.Close()
		return nil, fmt.Errorf("configure tun: %w", err)
	}

	return t, nil
}

// configure assigns the IP address to the Wintun adapter using netsh.
func (t *Interface) configure() error {
	// netsh requires the exact interface name as shown in "netsh interface show interface"
	// The Wintun adapter name from tun.Device.Name() is typically "ycair"
	cmd := exec.Command("netsh", "interface", "ip", "set", "address",
		fmt.Sprintf("name=%s", t.name),
		"source=static",
		fmt.Sprintf("addr=%s", t.ip),
		"mask=255.255.255.0",
		"gateway=none",
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		// netsh may report success via stderr on some Windows versions
		outStr := strings.TrimSpace(string(out))
		if outStr != "" && !strings.Contains(outStr, "Ok") {
			return fmt.Errorf("netsh set address: %w\n%s", err, outStr)
		}
	}

	return nil
}

// Name returns the Wintun adapter name (e.g., "ycair").
func (t *Interface) Name() string { return t.name }

// IP returns the assigned IP address.
func (t *Interface) IP() string { return t.ip }

// Read reads a raw IP packet from the Wintun TUN interface.
// Uses the underlying OS file descriptor for simple single-buffer reads.
func (t *Interface) Read(packet []byte) (int, error) {
	return t.dev.File().Read(packet)
}

// Write writes a raw IP packet to the Wintun TUN interface.
func (t *Interface) Write(packet []byte) (int, error) {
	return t.dev.File().Write(packet)
}

// Close shuts down the Wintun TUN adapter and releases the driver handle.
func (t *Interface) Close() error {
	return t.dev.Close()
}
