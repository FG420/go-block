package main

import (
	"bytes"
	"crypto/sha256"
	"fmt"
)

type (
	Block struct {
		Hash     []byte
		Data     []byte
		PrevHash []byte
	}

	BlockChain struct {
		blocks []*Block
	}
)

func (b *Block) DeriveHash() {
	info := bytes.Join([][]byte{b.Data, b.PrevHash}, []byte{})
	hash := sha256.Sum256(info)
	b.Hash = hash[:]
}

func CreateBlock(data string, prevHash []byte) *Block {
	block := &Block{[]byte{}, []byte(data), prevHash}
	block.DeriveHash()
	return block
}

func (bc *BlockChain) AddBlock(data string) {
	prevBlock := bc.blocks[len(bc.blocks)-1]
	newB := CreateBlock(data, prevBlock.Hash)
	bc.blocks = append(bc.blocks, newB)
}

func Genesis() *Block {
	return CreateBlock("Genesis", []byte{})
}

func InitBlockChain() *BlockChain {
	return &BlockChain{[]*Block{Genesis()}}
}

func main() {
	chain := InitBlockChain()

	chain.AddBlock("one Gen")
	chain.AddBlock("two Gen")
	chain.AddBlock("three Gen")

	for _, b := range chain.blocks {
		fmt.Printf("Previous Hash: %x\n", b.PrevHash)
		fmt.Printf("Data: %s\n", b.Data)
		fmt.Printf("Hash: %x\n\n", b.Hash)
	}
}
