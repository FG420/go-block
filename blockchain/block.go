package blockchain

type (
	Block struct {
		Hash     []byte
		Data     []byte
		PrevHash []byte
		Nonce    int
	}

	BlockChain struct {
		Blocks []*Block
	}
)

func CreateBlock(data string, prevHash []byte) *Block {
	block := &Block{[]byte{}, []byte(data), prevHash, 0}
	pow := NewProof(block)
	nonce, hash := pow.Run()

	block.Hash = hash
	block.Nonce = nonce
	return block
}

func (bc *BlockChain) AddBlock(data string) {
	prevBlock := bc.Blocks[len(bc.Blocks)-1]
	newB := CreateBlock(data, prevBlock.Hash)
	bc.Blocks = append(bc.Blocks, newB)
}

func Genesis() *Block {
	return CreateBlock("Genesis", []byte{})
}

func InitBlockChain() *BlockChain {
	return &BlockChain{[]*Block{Genesis()}}
}
