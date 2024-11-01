package blockchain

import (
	"github.com/FG420/go-block/handlers"
	"github.com/dgraph-io/badger"
)

type BlockChainIterator struct {
	CurrentHash []byte
	Database    *badger.DB
}

func (bc *BlockChain) Iterator() *BlockChainIterator {
	iter := &BlockChainIterator{
		bc.LastHash, bc.Database,
	}
	return iter
}

func (iter *BlockChainIterator) Next() *Block {
	var block *Block
	err := iter.Database.View(func(txn *badger.Txn) error {
		item, err := txn.Get(iter.CurrentHash)
		handlers.HandleErr(err)

		err = item.Value(func(val []byte) error {
			block = Deserialize(val)
			return nil
		})

		return nil
	})
	handlers.HandleErr(err)

	iter.CurrentHash = block.PrevHash
	return block

}
