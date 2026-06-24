//go:build linux

package tun

import (
	"fmt"
	"os/exec"

	"github.com/songgao/water"
)

// Interface is a TUN virtual network interface backed by the water library.
// On Linux, it creates a /dev/net/tun interface configured via iproute2.
type Interface struct {
	ifce *water.Interface
	name string
	ip   string
}

// Create creates a new TUN interface with the given IP address.
func Create(ip string) (*Interface, error) {
	config := water.Config{
		DeviceType: water.TUN,
	}

	ifce, err := water.New(config)
	if err != nil {
		return nil, fmt.Errorf("create tun: %w", err)
	}

	t := &Interface{
		ifce: ifce,
		name: ifce.Name(),
		ip:   ip,
	}

	if err := t.configure(); err != nil {
		ifce.Close()
		return nil, fmt.Errorf("configure tun: %w", err)
	}

	return t, nil
}

// configure sets the IP address and brings the interface up.
// On Linux, routing for 10.99.0.0/24 is handled automatically via the
// assigned IP/subnet because the kernel treats it as a directly connected network.
func (t *Interface) configure() error {
	// Assign IP with /24 subnet
	cmd := exec.Command("ip", "addr", "add", fmt.Sprintf("%s/24", t.ip), "dev", t.name)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("ip addr: %w\n%s", err, string(out))
	}

	// Bring interface up
	linkUp := exec.Command("ip", "link", "set", t.name, "up")
	if out, err := linkUp.CombinedOutput(); err != nil {
		return fmt.Errorf("ip link up: %w\n%s", err, string(out))
	}

	return nil
}

// Name returns the OS-assigned interface name (e.g., "tun0").
func (t *Interface) Name() string { return t.name }

// IP returns the assigned IP address.
func (t *Interface) IP() string { return t.ip }

// Read reads a raw IP packet from the TUN interface.
func (t *Interface) Read(packet []byte) (int, error) { return t.ifce.Read(packet) }

// Write writes a raw IP packet to the TUN interface.
func (t *Interface) Write(packet []byte) (int, error) { return t.ifce.Write(packet) }

// Close shuts down the TUN interface.
func (t *Interface) Close() error { return t.ifce.Close() }
