package wallet

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/json"
	"math/big"

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

func (w *Wallet) MarshalJSON() ([]byte, error) {
	mapStringAny := map[string]any{
		"PrivateKey": map[string]any{
			"D": w.PrivateKey.D,
			"PublicKey": map[string]any{
				"X": w.PrivateKey.PublicKey.X,
				"Y": w.PrivateKey.PublicKey.Y,
			},
			"Curve": w.PrivateKey.PublicKey.Curve.Params(),
		},
		"PublicKey": w.PublicKey,
	}
	return json.Marshal(mapStringAny)
}

func (w *Wallet) UnmarshalJSON(data []byte) error {
	var aux struct {
		PrivateKey struct {
			D         *big.Int `json:"D"`
			PublicKey struct {
				X     *big.Int       `json:"X"`
				Y     *big.Int       `json:"Y"`
				Curve elliptic.Curve `json:"Curve"`
			} `json:"PublicKey"`
		} `json:"PrivateKey"`
		PublicKey []byte `json:"PublicKey"`
	}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	w.PublicKey = aux.PublicKey

	w.PrivateKey = &ecdsa.PrivateKey{
		D: aux.PrivateKey.D,
		PublicKey: ecdsa.PublicKey{
			Curve: elliptic.P256(),
			X:     aux.PrivateKey.PublicKey.X,
			Y:     aux.PrivateKey.PublicKey.Y,
		},
	}

	return nil
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
