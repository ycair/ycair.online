package crypto

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/hex"

	"golang.org/x/crypto/pbkdf2"
)

const (
	pbkdf2Iterations = 100_000
	keySeedLen       = 32
)

func DeriveKeyPair(password, salt string) (pubKey string, privKey string, err error) {
	seed := pbkdf2.Key([]byte(password), []byte(salt), pbkdf2Iterations, ed25519.SeedSize, sha256.New)
	priv := ed25519.NewKeyFromSeed(seed)
	pub := priv.Public().(ed25519.PublicKey)
	return hex.EncodeToString(pub), hex.EncodeToString(priv), nil
}
