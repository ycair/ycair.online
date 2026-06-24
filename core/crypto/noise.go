package crypto

import (
	"fmt"
	"net"
	"time"

	"github.com/flynn/noise"
)

const maxHandshakeSize = 1024

type SecureChannel struct {
	sendCipher *noise.CipherState
	recvCipher *noise.CipherState
}

func PerformHandshake(conn *net.UDPConn, peerAddr *net.UDPAddr, isInitiator bool) (*SecureChannel, error) {
	cs := noise.NewCipherSuite(noise.DH25519, noise.CipherChaChaPoly, noise.HashBLAKE2s)

	hs, err := noise.NewHandshakeState(noise.Config{
		CipherSuite: cs,
		Pattern:     noise.HandshakeNN,
		Initiator:   isInitiator,
	})
	if err != nil {
		return nil, fmt.Errorf("new handshake: %w", err)
	}

	if isInitiator {
		return initiatorHandshake(hs, conn, peerAddr)
	}
	return responderHandshake(hs, conn, peerAddr)
}

func initiatorHandshake(hs *noise.HandshakeState, conn *net.UDPConn, peerAddr *net.UDPAddr) (*SecureChannel, error) {
	msg1, _, _, err := hs.WriteMessage(nil, nil)
	if err != nil {
		return nil, fmt.Errorf("build msg1: %w", err)
	}

	if _, err := conn.WriteTo(msg1, peerAddr); err != nil {
		return nil, fmt.Errorf("send msg1: %w", err)
	}

	buf := make([]byte, maxHandshakeSize)
	for {
		conn.SetReadDeadline(time.Now().Add(10 * time.Second))
		n, _, err := conn.ReadFromUDP(buf)
		if err != nil {
			return nil, fmt.Errorf("recv msg2: %w", err)
		}

		_, csSend, csRecv, hsErr := hs.ReadMessage(nil, buf[:n])
		if hsErr != nil {
			continue
		}

		conn.SetReadDeadline(time.Time{})
		return &SecureChannel{sendCipher: csSend, recvCipher: csRecv}, nil
	}
}

func responderHandshake(hs *noise.HandshakeState, conn *net.UDPConn, peerAddr *net.UDPAddr) (*SecureChannel, error) {
	buf := make([]byte, maxHandshakeSize)
	for {
		conn.SetReadDeadline(time.Now().Add(10 * time.Second))
		n, _, err := conn.ReadFromUDP(buf)
		if err != nil {
			return nil, fmt.Errorf("recv msg1: %w", err)
		}

		_, _, _, hsErr := hs.ReadMessage(nil, buf[:n])
		if hsErr != nil {
			continue
		}
		break
	}

	msg2, csRecv, csSend, err := hs.WriteMessage(nil, nil)
	if err != nil {
		return nil, fmt.Errorf("build msg2: %w", err)
	}

	if _, err := conn.WriteTo(msg2, peerAddr); err != nil {
		return nil, fmt.Errorf("send msg2: %w", err)
	}

	conn.SetReadDeadline(time.Time{})
	return &SecureChannel{sendCipher: csSend, recvCipher: csRecv}, nil
}

func (sc *SecureChannel) Encrypt(plaintext []byte) ([]byte, error) {
	if sc.sendCipher == nil {
		return nil, fmt.Errorf("no send cipher")
	}
	return sc.sendCipher.Encrypt(nil, nil, plaintext)
}

func (sc *SecureChannel) Decrypt(ciphertext []byte) ([]byte, error) {
	if sc.recvCipher == nil {
		return nil, fmt.Errorf("no receive cipher")
	}
	return sc.recvCipher.Decrypt(nil, nil, ciphertext)
}

func (sc *SecureChannel) Overhead() int {
	return 16 + 16
}
