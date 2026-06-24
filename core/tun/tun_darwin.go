package tun

import (
	"fmt"
	"os/exec"
)

func (t *Interface) configure() error {
	cmd := exec.Command("ifconfig", t.name, t.ip, t.ip, "netmask", "255.255.255.0", "up")
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("ifconfig %s: %w\n%s", t.name, err, string(out))
	}

	addRoute := exec.Command("route", "add", "-net", "10.99.0.0/24", "-interface", t.name)
	if out, err := addRoute.CombinedOutput(); err != nil {
		return fmt.Errorf("route add: %w\n%s", err, string(out))
	}

	return nil
}
