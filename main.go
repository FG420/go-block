package main

import (
	"fmt"
	"strconv"

	"github.com/FG420/go-block/blockchain"
)

func main() {
	chain := blockchain.InitBlockChain()

	chain.AddBlock("1st block after Gen")
	chain.AddBlock("2nd block after Gen")
	chain.AddBlock("3rd block after Gen")

	for _, b := range chain.Blocks {
		fmt.Printf("Previous Hash: %x\n", b.PrevHash)
		fmt.Printf("Data: %s\n", b.Data)
		fmt.Printf("Hash: %x\n", b.Hash)

		pow := blockchain.NewProof(b)
		fmt.Printf("PoW: %s\n\n", strconv.FormatBool(pow.Validate()))
	}
}
