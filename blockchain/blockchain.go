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

func (bc *BlockChain) AddBlock(txs []*Transaction) {
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

	newBlock := CreateBlock(txs, lastHash)

	err = bc.Database.Update(func(txn *badger.Txn) error {
		err = txn.Set(newBlock.Hash, newBlock.Serialize())
		handlers.HandleErr(err)

		err = txn.Set([]byte("lh"), newBlock.Hash)
		bc.LastHash = newBlock.Hash

		return nil
	})
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

func (bc *BlockChain) FindUnspentTx(pubKeyHash []byte) []Transaction {
	var unspentTxs []Transaction
	spentTx0s := make(map[string][]int)
	iter := bc.Iterator()

	for {
		block := iter.Next()

		for _, tx := range block.Transactions {
			txId := hex.EncodeToString(tx.ID)

		Outputs:
			for outIdx, out := range tx.Outputs {
				if spentTx0s[txId] != nil {
					for _, spentOut := range spentTx0s[txId] {
						if spentOut == outIdx {
							continue Outputs
						}
					}
				}
				if out.IsLockedWithKey(pubKeyHash) {
					unspentTxs = append(unspentTxs, *tx)
				}
			}

			if !tx.IsCoinbase() {
				for _, in := range tx.Inputs {
					if in.UsesKey(pubKeyHash) {
						inTxId := hex.EncodeToString(in.ID)
						spentTx0s[inTxId] = append(spentTx0s[inTxId], in.Out)
					}
				}
			}

		}

		if len(block.PrevHash) == 0 {
			break
		}
	}
	return unspentTxs
}

func (bc *BlockChain) FindUTxO(pubKeyHash []byte) []TxOutput {
	var UTxOs []TxOutput

	unspentTxs := bc.FindUnspentTx(pubKeyHash)

	for _, tx := range unspentTxs {
		for _, out := range tx.Outputs {
			if out.IsLockedWithKey(pubKeyHash) {
				UTxOs = append(UTxOs, out)
			}
		}
	}

	return UTxOs
}

func (bc *BlockChain) FindSpendableOutputs(pubKeyHash []byte, amount int) (int, map[string][]int) {
	unspentOuts := make(map[string][]int)
	unspentTxs := bc.FindUnspentTx(pubKeyHash)
	accumulated := 0

Work:
	for _, tx := range unspentTxs {
		txId := hex.EncodeToString(tx.ID)

		for outIdx, out := range tx.Outputs {
			if out.IsLockedWithKey(pubKeyHash) && accumulated < amount {
				accumulated += out.Value
				unspentOuts[txId] = append(unspentOuts[txId], outIdx)

				if accumulated >= amount {
					break Work
				}
			}
		}
	}

	return accumulated, unspentOuts
}

func (bc *BlockChain) FincTransaction(id []byte) (Transaction, error) {
	iter := bc.Iterator()

	for {
		block := iter.Next()

		for _, tx := range block.Transactions {
			if bytes.Compare(tx.ID, id) == 0 {
				return *tx, nil
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
		prevTx, err := bc.FincTransaction(in.ID)
		handlers.HandleErr(err)
		prevTxs[hex.EncodeToString(prevTx.ID)] = prevTx
	}

	tx.Sign(privKey, prevTxs)
}

func (bc *BlockChain) VerifyTransaction(tx *Transaction) bool {
	prevTxs := make(map[string]Transaction)

	for _, in := range tx.Inputs {
		prevTx, err := bc.FincTransaction(in.ID)
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
		fmt.Println("No existring blockchain found, created one!")
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
