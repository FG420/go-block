package blockchain

import (
	"bytes"
	"crypto/ecdsa"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"runtime"

	"github.com/FG420/go-block/handlers"
	"github.com/dgraph-io/badger"
)

type (
	BlockChain struct {
		LastHash []byte
		Database *badger.DB
	}
	BlockChainIterator struct {
		CurrentHash []byte
		Database    *badger.DB
	}
)

func (bc *BlockChain) AddBlock(txs []*Transaction) *Block {
	var lastHash []byte

	err := bc.Database.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte("lh"))
		handlers.HandleErr(err)
		err = item.Value(func(val []byte) error {
			lastHash = append(lastHash, val...)
			return nil
		})
		return nil
	})
	handlers.HandleErr(err)

	newBlock := CreateBlock(txs, lastHash)

	err = bc.Database.Update(func(txn *badger.Txn) error {
		err = txn.Set(newBlock.Hash, newBlock.Serialize())
		handlers.HandleErr(err)

		err = txn.Set([]byte("lh"), newBlock.Hash)
		bc.LastHash = newBlock.Hash

		return nil
	})
	handlers.HandleErr(err)

	return newBlock
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

func (bc *BlockChain) FindUTxO() map[string]TxOutputs {
	utxo := make(map[string]TxOutputs)
	spentTxos := make(map[string][]int)
	iter := bc.Iterator()

	for {
		block := iter.Next()
		for _, tx := range block.Transactions {
			txID := hex.EncodeToString(tx.ID)

		Outputs:
			for outIdx, out := range tx.Outputs {
				if spentTxos[txID] != nil {
					for _, spentOut := range spentTxos[txID] {
						if spentOut == outIdx {
							continue Outputs
						}
					}
				}

				outs := utxo[txID]
				outs.Outputs = append(outs.Outputs, out)
				utxo[txID] = outs
			}

			if !tx.IsCoinbase() {
				for _, in := range tx.Inputs {
					inTxID := hex.EncodeToString(in.ID)
					spentTxos[inTxID] = append(spentTxos[inTxID], in.Out)
				}
			}
		}

		if len(block.PrevHash) == 0 {
			break
		}
	}

	return utxo
}

func (bc *BlockChain) FindTransaction(id []byte) (Transaction, error) {
	iter := bc.Iterator()

	for {
		block := iter.Next()
		for _, tx := range block.Transactions {
			for _, in := range tx.Inputs {
				if bytes.Compare(id, in.ID) == 1 {
					return *tx, nil
				}
			}
		}

		if len(block.PrevHash) == 0 {
			break
		}
	}

	return Transaction{}, errors.New("Transaction doesn't exist")
}

func (bc *BlockChain) SignTransaction(tx *Transaction, privKey ecdsa.PrivateKey) {
	prevTxs := make(map[string]Transaction)

	for _, in := range tx.Inputs {
		prevTx, err := bc.FindTransaction(in.ID)
		handlers.HandleErr(err)
		prevTxs[hex.EncodeToString(prevTx.ID)] = prevTx
	}

	log.Println("transaction Signed")
	tx.Sign(privKey, prevTxs)
}

func (bc *BlockChain) VerifyTransaction(tx *Transaction) bool {
	if tx.IsCoinbase() {
		return true
	}
	prevTxs := make(map[string]Transaction)

	for _, in := range tx.Inputs {
		prevTx, err := bc.FindTransaction(in.ID)
		handlers.HandleErr(err)
		prevTxs[hex.EncodeToString(prevTx.ID)] = prevTx
	}

	return tx.Verify(prevTxs)
}

func InitBlockChain(addr string) *BlockChain {
	var lastHash []byte
	if handlers.DbExist() {
		fmt.Println("Blockchain already exists")
		runtime.Goexit()
	}

	opt := badger.DefaultOptions(handlers.DbPath)
	db, err := badger.Open(opt)
	handlers.HandleErr(err)

	err = db.Update(func(txn *badger.Txn) error {
		cbtx := CoinbaseTx(addr, handlers.GenesisData)
		genesis := Genesis(cbtx)
		log.Println("Genesis created")
		err = txn.Set(genesis.Hash, genesis.Serialize())
		handlers.HandleErr(err)
		err = txn.Set([]byte("lh"), genesis.Hash)

		lastHash = genesis.Hash
		return err
	})
	handlers.HandleErr(err)
	blockchain := BlockChain{lastHash, db}
	return &blockchain
}

func ContinueBlockChain(addr string) *BlockChain {
	if !handlers.DbExist() {
		fmt.Println("No existring blockchain found, create one!")
		runtime.Goexit()
	}
	var lastHash []byte

	opt := badger.DefaultOptions(handlers.DbPath)
	db, err := badger.Open(opt)
	handlers.HandleErr(err)

	err = db.Update(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte("lh"))
		handlers.HandleErr(err)

		err = item.Value(func(val []byte) error {
			lastHash = append(lastHash, val...)
			return nil
		})
		return err
	})
	handlers.HandleErr(err)

	chain := BlockChain{lastHash, db}
	return &chain
}
