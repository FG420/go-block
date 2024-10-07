package blockchain

import (
	"encoding/hex"
	"fmt"
	"log"
	"runtime"

	"github.com/FG420/go-block/utils"
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
		utils.HandleErr(err)
		err = item.Value(func(val []byte) error {
			lastHash = append(lastHash, val...)
			return nil
		})
		return nil
	})

	newBlock := CreateBlock(txs, lastHash)

	err = bc.Database.Update(func(txn *badger.Txn) error {
		err = txn.Set(newBlock.Hash, newBlock.Serialize())
		utils.HandleErr(err)

		err = txn.Set([]byte("lh"), newBlock.Hash)
		bc.LastHash = newBlock.Hash

		return nil
	})
}

func InitBlockChain(addr string) *BlockChain {
	var lastHash []byte
	if utils.DbExist() {
		fmt.Println("Blockchain already exists")
		runtime.Goexit()
	}

	opt := badger.DefaultOptions(utils.DbPath)
	db, err := badger.Open(opt)
	utils.HandleErr(err)

	err = db.Update(func(txn *badger.Txn) error {
		cbtx := CoinbaseTx(addr, utils.GenesisData)
		genesis := Genesis(cbtx)
		log.Println("Genesis created")
		err = txn.Set(genesis.Hash, genesis.Serialize())
		utils.HandleErr(err)
		err = txn.Set([]byte("lh"), genesis.Hash)

		lastHash = genesis.Hash
		return err
	})
	utils.HandleErr(err)
	blockchain := BlockChain{lastHash, db}
	return &blockchain
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
		utils.HandleErr(err)

		err = item.Value(func(val []byte) error {
			block = Deserialize(val)
			return nil
		})

		return nil
	})
	utils.HandleErr(err)

	iter.CurrentHash = block.PrevHash
	return block

}

func ContinueBlockChain(addr string) *BlockChain {
	if !utils.DbExist() {
		fmt.Println("No existring blockchain found, created one!")
	}
	var lastHash []byte

	opt := badger.DefaultOptions(utils.DbPath)
	db, err := badger.Open(opt)
	utils.HandleErr(err)

	err = db.Update(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte("lh"))
		utils.HandleErr(err)

		err = item.Value(func(val []byte) error {
			lastHash = append(lastHash, val...)
			return nil
		})
		return err
	})
	utils.HandleErr(err)

	chain := BlockChain{lastHash, db}
	return &chain
}

func (bc *BlockChain) FindUnspentTx(addr string) []Transaction {
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
				if out.CanBeUnlocked(addr) {
					unspentTxs = append(unspentTxs, *tx)
				}
			}

			if !tx.IsCoinbase() {
				for _, in := range tx.Inputs {
					if in.CanUnlock(addr) {
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

func (bc *BlockChain) FindUTxO(addr string) []TxOutput {
	var UTxOs []TxOutput

	unspentTxs := bc.FindUnspentTx(addr)

	for _, tx := range unspentTxs {
		for _, out := range tx.Outputs {
			if out.CanBeUnlocked(addr) {
				UTxOs = append(UTxOs, out)
			}
		}
	}

	return UTxOs
}

func (bc *BlockChain) FindSpendableOutputs(addr string, amount int) (int, map[string][]int) {
	unspentOuts := make(map[string][]int)
	unspentTxs := bc.FindUnspentTx(addr)
	accumulated := 0

Work:
	for _, tx := range unspentTxs {
		txId := hex.EncodeToString(tx.ID)

		for outIdx, out := range tx.Outputs {
			if out.CanBeUnlocked(addr) && accumulated < amount {
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
