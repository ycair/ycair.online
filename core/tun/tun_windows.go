package tun

import "fmt"

func (t *Interface) configure() error {
	return fmt.Errorf("tun: Windows requires wintun driver (install from wireguard.com)")
}
