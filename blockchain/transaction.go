package blockchain

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/gob"
	"encoding/hex"
	"fmt"
	"log"
	"math/big"
	"strings"

	"github.com/FG420/go-block/handlers"
	"github.com/FG420/go-block/wallet"
)

type (
	Transaction struct {
		ID      []byte
		Inputs  []TxInput
		Outputs []TxOutput
	}
)

func (tx *Transaction) String() string {
	var lines []string

	lines = append(lines, fmt.Sprintf("-- Transaction %x: \n", tx.ID))
	for i, input := range tx.Inputs {
		lines = append(lines, fmt.Sprintf("	Input: %d", i))
		lines = append(lines, fmt.Sprintf("		TxID: %x", input.ID))
		lines = append(lines, fmt.Sprintf("		Out: %d", input.Out))
		lines = append(lines, fmt.Sprintf("		Signature: %x", input.Signature))
		lines = append(lines, fmt.Sprintf("		PubKey: %x", input.PubKey))
	}

	for i, output := range tx.Outputs {
		lines = append(lines, fmt.Sprintf("	Output %d: ", i))
		lines = append(lines, fmt.Sprintf("		Value: %d ", output.Value))
		lines = append(lines, fmt.Sprintf("		Script: %x ", output.PubKeyHash))
	}

	return strings.Join(lines, "\n")
}

// func (tx *Transaction) SetID() {
// 	var encoded bytes.Buffer
// 	var hash [32]byte

// 	encode := gob.NewEncoder(&encoded)
// 	err := encode.Encode(tx)
// 	handlers.HandleErr(err)

// 	hash = sha256.Sum256(encoded.Bytes())
// 	tx.ID = hash[:]
// }

func (tx *Transaction) IsCoinbase() bool {
	return len(tx.Inputs) == 1 &&
		len(tx.Inputs[0].ID) == 0 &&
		tx.Inputs[0].Out == -1
}

func (tx *Transaction) Serialize() []byte {
	var encoded bytes.Buffer

	enc := gob.NewEncoder(&encoded)
	err := enc.Encode(tx)
	handlers.HandleErr(err)

	return encoded.Bytes()
}

func (tx *Transaction) Hash() []byte {
	var hash [32]byte

	txCopy := *tx
	hash = sha256.Sum256(txCopy.Serialize())
	txCopy.ID = hash[:]
	return hash[:]
}

func (tx *Transaction) TrimmedCopy() *Transaction {
	var inputs []TxInput
	var outputs []TxOutput

	for _, in := range tx.Inputs {
		inputs = append(inputs, TxInput{in.ID, in.Out, nil, nil})
	}

	for _, out := range tx.Outputs {
		outputs = append(outputs, out)
	}

	return &Transaction{
		tx.ID,
		inputs,
		outputs,
	}
}

func (tx *Transaction) Sign(privKey ecdsa.PrivateKey, prevTxs map[string]Transaction) {
	if tx.IsCoinbase() {
		log.Println("Coinbase transaction, no signing needed.")
		return
	}

	for _, in := range tx.Inputs {
		if _, exists := prevTxs[hex.EncodeToString(in.ID)]; !exists {
			log.Panic("ERROR: Previous Transaction deasn't exist")
		}
	}

	txCopy := tx.TrimmedCopy()

	for inId, in := range txCopy.Inputs {
		prevTX := prevTxs[hex.EncodeToString(in.ID)]
		txCopy.Inputs[inId].Signature = nil
		txCopy.Inputs[inId].PubKey = prevTX.Outputs[in.Out].PubKeyHash

		dataToSign := fmt.Sprintf("%x\n", txCopy)

		r, s, err := ecdsa.Sign(rand.Reader, &privKey, []byte(dataToSign))
		handlers.HandleErr(err)
		signature := append(r.Bytes(), s.Bytes()...)

		tx.Inputs[inId].Signature = signature
		txCopy.Inputs[inId].PubKey = nil
	}
}

func (tx *Transaction) Verify(prevTxs map[string]Transaction) bool {
	if tx.IsCoinbase() {
		return true
	}

	for _, in := range tx.Inputs {
		if _, exists := prevTxs[hex.EncodeToString(in.ID)]; !exists {
			log.Panic("ERROR: Previous Transaction deasn't exist")
		}
	}

	txCopy := tx.TrimmedCopy()
	curve := elliptic.P256()

	for inId, in := range tx.Inputs {
		prevTx := prevTxs[hex.EncodeToString(in.ID)]
		txCopy.Inputs[inId].Signature = nil
		txCopy.Inputs[inId].PubKey = prevTx.Outputs[in.Out].PubKeyHash

		r := big.Int{}
		s := big.Int{}

		sigLen := len(in.Signature)
		r.SetBytes(in.Signature[:(sigLen / 2)])
		s.SetBytes(in.Signature[(sigLen / 2):])

		x := big.Int{}
		y := big.Int{}
		keyLen := len(in.PubKey)
		x.SetBytes(in.PubKey[:(keyLen / 2)])
		y.SetBytes(in.PubKey[(keyLen / 2):])

		dataToVerify := fmt.Sprintf("%x\n", txCopy)

		rawPubKey := ecdsa.PublicKey{Curve: curve, X: &x, Y: &y}
		if ecdsa.Verify(&rawPubKey, []byte(dataToVerify), &r, &s) == false {
			return false
		}
		txCopy.Inputs[inId].PubKey = nil
	}

	return true
}

func CoinbaseTx(to, data string) *Transaction {
	if data == "" {
		randData := make([]byte, 24)
		_, err := rand.Read(randData)
		handlers.HandleErr(err)
		data = fmt.Sprintf("%x", randData)
	}

	txin := TxInput{[]byte{}, -1, nil, []byte(data)}
	txout := NewTxOutput(100, to)

	tx := Transaction{nil, []TxInput{txin}, []TxOutput{*txout}}
	tx.Hash()

	return &tx
}

func NewTransaction(w *wallet.Wallet, to string, amount int, utxo *UTXOSet) *Transaction {
	var inputs []TxInput
	var outputs []TxOutput

	pubKeyHash := wallet.PublicKeyHash(w.PublicKey)
	acc, validOutputs := utxo.FindSpendableOutputs(pubKeyHash, amount)

	if acc < amount {
		log.Panic("Error: not enough funds")
	}

	for txid, outs := range validOutputs {
		txID, err := hex.DecodeString(txid)
		handlers.HandleErr(err)

		for _, out := range outs {
			newInput := TxInput{txID, out, nil, w.PublicKey}
			inputs = append(inputs, newInput)
		}
	}

	from := fmt.Sprintf("%s", w.Address())

	outputs = append(outputs, *NewTxOutput(amount, to))
	if acc > amount {
		outputs = append(outputs, *NewTxOutput(acc-amount, from))
	}

	tx := Transaction{nil, inputs, outputs}
	tx.ID = tx.Hash()
	utxo.BlockChain.SignTransaction(&tx, *w.PrivateKey)

	return &tx
}
