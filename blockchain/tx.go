package blockchain

import (
	"bytes"
	"encoding/gob"

	"github.com/FG420/go-block/handlers"
	"github.com/FG420/go-block/wallet"
)

type (
	TxInput struct {
		ID        []byte
		Out       int
		Signature []byte
		PubKey    []byte
	}

	TxOutput struct {
		Value      int
		PubKeyHash []byte
	}

	TxOutputs struct {
		Outputs []TxOutput
	}
)

func (in *TxInput) UsesKey(pubKeyHash []byte) bool {
	lockingHash := wallet.PublicKeyHash(in.PubKey)
	return bytes.Compare(lockingHash, pubKeyHash) == 0
}

func (out *TxOutput) Lock(addr []byte) {
	pubKeyHash := wallet.Base58Decode(addr)
	pubKeyHash = pubKeyHash[1 : len(pubKeyHash)-4] // taking away Version and Checksum Hash
	out.PubKeyHash = pubKeyHash
}

func (out *TxOutput) IsLockedWithKey(pubKeyHash []byte) bool {
	return bytes.Compare(out.PubKeyHash, pubKeyHash) == 0
}

func (outs *TxOutputs) Serialize() []byte {
	var buffer bytes.Buffer

	enc := gob.NewEncoder(&buffer)
	err := enc.Encode(outs)
	handlers.HandleErr(err)

	return buffer.Bytes()
}

func DeserializeOuts(data []byte) TxOutputs {
	var outputs TxOutputs

	dec := gob.NewDecoder(bytes.NewReader(data))
	err := dec.Decode(&outputs)
	handlers.HandleErr(err)

	return outputs
}

func NewTxOutput(value int, addr string) *TxOutput {
	txO := &TxOutput{value, nil}
	txO.Lock([]byte(addr))

	return txO
}
