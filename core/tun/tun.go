package tun

import (
	"fmt"
	"io"

	"github.com/songgao/water"
)

const MTU = 1500

type Interface struct {
	ifce *water.Interface
	name string
	ip   string
}

func Create(ip string) (*Interface, error) {
	config := water.Config{
		DeviceType: water.TUN,
	}

	ifce, err := water.New(config)
	if err != nil {
		return nil, fmt.Errorf("create tun: %w", err)
	}

	tun := &Interface{
		ifce: ifce,
		name: ifce.Name(),
		ip:   ip,
	}

	if err := tun.configure(); err != nil {
		ifce.Close()
		return nil, fmt.Errorf("configure tun: %w", err)
	}

	return tun, nil
}

func (t *Interface) Name() string {
	return t.name
}

func (t *Interface) IP() string {
	return t.ip
}

func (t *Interface) Read(packet []byte) (int, error) {
	return t.ifce.Read(packet)
}

func (t *Interface) Write(packet []byte) (int, error) {
	return t.ifce.Write(packet)
}

func (t *Interface) ReadPacket() ([]byte, error) {
	buf := make([]byte, MTU)
	n, err := t.ifce.Read(buf)
	if err != nil {
		return nil, err
	}
	return buf[:n], nil
}

func (t *Interface) WritePacket(packet []byte) error {
	_, err := t.ifce.Write(packet)
	return err
}

func (t *Interface) Close() error {
	return t.ifce.Close()
}

func (t *Interface) ReadFrom(r io.Reader) error {
	buf := make([]byte, MTU)
	for {
		n, err := r.Read(buf)
		if err != nil {
			return err
		}
		if _, err := t.ifce.Write(buf[:n]); err != nil {
			return err
		}
	}
}
