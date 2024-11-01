package blockchain

import (
	"bytes"
	"crypto/ecdsa"
	"encoding/gob"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"runtime"

	"github.com/FG420/go-block/handlers"
	"github.com/dgraph-io/badger"
)

type BlockChain struct {
	LastHash []byte
	Database *badger.DB
}

func (bc *BlockChain) AddBlock(block *Block) {
	var lastHash []byte
	var lastBlockData []byte

	err := bc.Database.Update(func(txn *badger.Txn) error {
		if _, err := txn.Get(block.Hash); err == nil {
			return nil
		}

		blockData := block.Serialize()
		err := txn.Set(block.Hash, blockData)
		handlers.HandleErr(err)

		item, err := txn.Get([]byte("lh"))
		item.Value(func(val []byte) error {
			lastHash = val
			return nil
		})

		item, err = txn.Get([]byte(lastHash))
		handlers.HandleErr(err)

		item.Value(func(val []byte) error {
			lastBlockData = val
			return nil
		})

		lastBlock := Deserialize(lastBlockData)

		if block.Height > lastBlock.Height {
			err = txn.Set([]byte("lh"), block.Hash)
			handlers.HandleErr(err)
			bc.LastHash = block.Hash
		}

		return nil
	})
	handlers.HandleErr(err)
}

func (bc *BlockChain) MineBlock(txs []*Transaction) *Block {
	var lastHash []byte
	var lastBlockData []byte
	var lastHeight int

	for _, tx := range txs {
		if bc.VerifyTransaction(tx) != true {
			log.Panic("Invalid Transaction")
		}
	}

	err := bc.Database.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte("lh"))
		handlers.HandleErr(err)

		err = item.Value(func(val []byte) error {
			lastHash = val
			return err
		})
		handlers.HandleErr(err)

		item, err = txn.Get([]byte(lastHash))
		handlers.HandleErr(err)
		err = item.Value(func(val []byte) error {
			lastBlockData = val
			return err
		})
		handlers.HandleErr(err)

		lastBlock := Deserialize(lastBlockData)
		lastHeight = lastBlock.Height

		return err
	})
	handlers.HandleErr(err)

	newBlock := CreateBlock(txs, lastHash, lastHeight+1)

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

func (bc *BlockChain) GetBlock(blockHash []byte) (*Block, error) {
	var block Block
	var blockData []byte

	err := bc.Database.View(func(txn *badger.Txn) error {
		if item, err := txn.Get(blockHash); err != nil {
			return errors.New("Block not found")
		} else {
			item.Value(func(val []byte) error {
				blockData = val
				return nil
			})

			block = *Deserialize(blockData)
		}

		return nil
	})
	if err != nil {
		return &block, err
	}

	return &block, nil
}

func (bc *BlockChain) GetBlockHashes() [][]byte {
	var blocks [][]byte

	iter := bc.Iterator()

	for {
		block := iter.Next()
		blocks = append(blocks, block.Hash)
		if len(block.PrevHash) == 0 {
			break
		}
	}

	return blocks
}

func (bc *BlockChain) GetBestHeight() int {
	var lastBlock Block
	var lastHash []byte
	var lastBlockData []byte

	err := bc.Database.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte("lh"))
		handlers.HandleErr(err)
		item.Value(func(val []byte) error {
			lastHash = val
			return nil
		})

		item, err = txn.Get(lastHash)
		handlers.HandleErr(err)
		item.Value(func(val []byte) error {
			lastBlockData = val
			return nil
		})

		lastBlock = *Deserialize(lastBlockData)

		return nil
	})
	handlers.HandleErr(err)

	return lastBlock.Height
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
	log.Println(" id ->", id)

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
		log.Println(in.ID)
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

func DeserializeTransaction(data []byte) Transaction {
	var tx Transaction

	dec := gob.NewDecoder(bytes.NewReader(data))
	err := dec.Decode(&tx)
	handlers.HandleErr(err)

	return tx
}

func InitBlockChain(addr, nodeId string) *BlockChain {
	path := fmt.Sprintf(handlers.DbPath, nodeId)

	if handlers.DbExist(path) {
		fmt.Println("Blockchain already exists")
		runtime.Goexit()
	}

	opt := badger.DefaultOptions(handlers.DbPath)
	opt.Dir = path
	opt.ValueDir = path

	db, err := handlers.OpenDB(path, opt)
	handlers.HandleErr(err)

	var lastHash []byte
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
	log.Println("Path -> ", path)

	return &blockchain
}

func ContinueBlockChain(nodeId string) *BlockChain {
	path := fmt.Sprintf(handlers.DbPath, nodeId)
	if !handlers.DbExist(path) {
		fmt.Println("No existring blockchain found, create one!")
		runtime.Goexit()
	}

	opt := badger.DefaultOptions(handlers.DbPath)
	opt.Dir = path
	opt.ValueDir = path

	db, err := handlers.OpenDB(path, opt)
	handlers.HandleErr(err)

	var lastHash []byte
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
