// Package tun provides a cross-platform TUN (Layer 3) virtual network interface.
//
// Platform backends:
//   - macOS:    water (utun) + ifconfig/route
//   - Linux:    water (/dev/net/tun) + ip addr/link
//   - Windows:  wireguard-go/tun (Wintun) + netsh
//
// All platforms export an identical Interface type with Read/Write for raw IP packets.
package tun

// MTU is the Maximum Transmission Unit for the virtual interface.
const MTU = 1500
