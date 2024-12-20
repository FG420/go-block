package blockchain

import (
	"bytes"
	"encoding/gob"
	"time"

	"github.com/FG420/go-block/handlers"
)

type (
	Block struct {
		Hash         []byte
		Transactions []*Transaction
		PrevHash     []byte
		Nonce        int
		Timestamp    int64
		Height       int
	}
)

func (b *Block) HashTransactions() []byte {
	var txHashes [][]byte

	for _, tx := range b.Transactions {
		txHashes = append(txHashes, tx.Serialize())
	}

	tree := NewMerkleTree(txHashes)
	// txHash = sha256.Sum256(bytes.Join(txHashes, []byte{}))

	return tree.RootNode.Data
}

func (b *Block) Serialize() []byte {
	var res bytes.Buffer
	encoder := gob.NewEncoder(&res)

	err := encoder.Encode(b)
	handlers.HandleErr(err)

	return res.Bytes()
}

func Deserialize(data []byte) *Block {
	var block Block

	decoder := gob.NewDecoder(bytes.NewReader(data))

	err := decoder.Decode(&block)
	handlers.HandleErr(err)

	return &block
}

func CreateBlock(txs []*Transaction, prevHash []byte, height int) *Block {
	block := &Block{[]byte{}, txs, prevHash, 0, time.Now().Unix(), height}
	pow := NewProof(block)
	nonce, hash := pow.Run()

	block.Hash = hash
	block.Nonce = nonce
	return block
}

func Genesis(coinbase *Transaction) *Block {
	return CreateBlock([]*Transaction{coinbase}, []byte{}, 0)
}
