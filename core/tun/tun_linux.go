package tun

import (
	"fmt"
	"os/exec"
)

func (t *Interface) configure() error {
	cmd := exec.Command("ip", "addr", "add", fmt.Sprintf("%s/24", t.ip), "dev", t.name)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("ip addr: %w\n%s", err, string(out))
	}

	linkUp := exec.Command("ip", "link", "set", t.name, "up")
	if out, err := linkUp.CombinedOutput(); err != nil {
		return fmt.Errorf("ip link up: %w\n%s", err, string(out))
	}

	return nil
}
