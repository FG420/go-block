package wallet

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"

	"github.com/FG420/go-block/handlers"
)

const (
	checksumLength = 4
	version        = byte(0x00)
)

type Wallet struct {
	PrivateKey *ecdsa.PrivateKey
	PublicKey  []byte
}

func (w *Wallet) Address() []byte {
	pubHash := PublicKeyHash(w.PublicKey)
	versionedHash := append([]byte{version}, pubHash...)
	checksum := Checksum(versionedHash)

	fullHash := append(versionedHash, checksum...)
	addr := Base58Encode(fullHash)

	// fmt.Printf("address: %x\n", addr)

	return addr
}

func ValidateAddress(addr string) bool {
	pubKeyHash := Base58Decode([]byte(addr))
	actualChecksum := pubKeyHash[len(pubKeyHash)-checksumLength:]

	version := pubKeyHash[0]
	pubKeyHash = pubKeyHash[1 : len(pubKeyHash)-checksumLength]
	targetChecksum := Checksum(append([]byte{version}, pubKeyHash...))

	return bytes.Compare(actualChecksum, targetChecksum) == 0
}

func NewPair() (*ecdsa.PrivateKey, []byte, error) {
	curve := elliptic.P256()
	private, err := ecdsa.GenerateKey(curve, rand.Reader)
	if err != nil {
		return nil, nil, err
	}

	pub := append(private.PublicKey.X.Bytes(), private.PublicKey.Y.Bytes()...)

	return private, pub, nil
}

func MakeWallet() *Wallet {
	private, public, _ := NewPair()
	return &Wallet{
		PrivateKey: private,
		PublicKey:  public,
	}
}

func PublicKeyHash(pubKey []byte) []byte {
	pubHash := sha256.Sum256(pubKey)

	hasher := sha256.New()
	_, err := hasher.Write(pubHash[:])
	handlers.HandleErr(err)

	pubSha := hasher.Sum(nil)
	return pubSha
}

func Checksum(payload []byte) []byte {
	firstHash := sha256.Sum256(payload)
	secondHash := sha256.Sum256(firstHash[:])

	return secondHash[:checksumLength]
}
