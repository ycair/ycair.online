//go:build windows

package tun

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"golang.org/x/sys/windows"
	"golang.zx2c4.com/wireguard/tun"
)

type Interface struct {
	dev  tun.Device
	name string
	ip   string
}

func Create(ip string) (*Interface, error) {
	if exe, err := os.Executable(); err == nil {
		dir := filepath.Dir(exe)
		if _, err := os.Stat(filepath.Join(dir, "wintun.dll")); err == nil {
			windows.SetDllDirectory(dir)
		}
	}

	dev, err := tun.CreateTUN("ycair", MTU)
	if err != nil {
		return nil, fmt.Errorf("create wintun tun: %w (run as administrator?)", err)
	}

	name, err := dev.Name()
	if err != nil {
		dev.Close()
		return nil, fmt.Errorf("get tun name: %w", err)
	}

	t := &Interface{dev: dev, name: name, ip: ip}

	if err := t.configure(); err != nil {
		dev.Close()
		return nil, fmt.Errorf("configure: %w", err)
	}

	return t, nil
}

func (t *Interface) configure() error {
	cmd := exec.Command("netsh", "interface", "ip", "set", "address",
		fmt.Sprintf("name=%s", t.name),
		"source=static",
		fmt.Sprintf("addr=%s", t.ip),
		"mask=255.255.255.0",
		"gateway=none",
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		outStr := strings.TrimSpace(string(out))
		if outStr != "" && !strings.Contains(outStr, "Ok") {
			return fmt.Errorf("netsh: %w\n%s", err, outStr)
		}
	}
	return nil
}

func (t *Interface) Name() string                    { return t.name }
func (t *Interface) IP() string                      { return t.ip }
func (t *Interface) Read(packet []byte) (int, error)  { return t.dev.File().Read(packet) }
func (t *Interface) Write(packet []byte) (int, error) { return t.dev.File().Write(packet) }
func (t *Interface) Close() error                     { return t.dev.Close() }
