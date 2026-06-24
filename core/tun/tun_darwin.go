//go:build darwin

package tun

import (
	"fmt"
	"os/exec"

	"github.com/songgao/water"
)

// Interface is a TUN virtual network interface backed by the water library.
// On macOS, it creates a utun interface configured via ifconfig and route.
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

// configure sets the IP address and adds a route for the mesh subnet.
func (t *Interface) configure() error {
	// Assign IP and bring interface up
	cmd := exec.Command("ifconfig", t.name, t.ip, t.ip, "netmask", "255.255.255.0", "up")
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("ifconfig %s: %w\n%s", t.name, err, string(out))
	}

	// Add route for the mesh network (10.99.0.0/24)
	addRoute := exec.Command("route", "add", "-net", "10.99.0.0/24", "-interface", t.name)
	if out, err := addRoute.CombinedOutput(); err != nil {
		return fmt.Errorf("route add: %w\n%s", err, string(out))
	}

	return nil
}

// Name returns the OS-assigned interface name (e.g., "utun3").
func (t *Interface) Name() string { return t.name }

// IP returns the assigned IP address.
func (t *Interface) IP() string { return t.ip }

// Read reads a raw IP packet from the TUN interface.
func (t *Interface) Read(packet []byte) (int, error) { return t.ifce.Read(packet) }

// Write writes a raw IP packet to the TUN interface.
func (t *Interface) Write(packet []byte) (int, error) { return t.ifce.Write(packet) }

// Close shuts down the TUN interface.
func (t *Interface) Close() error { return t.ifce.Close() }
