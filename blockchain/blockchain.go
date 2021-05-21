package blockchain

import (
	"fmt"

	"github.com/dgraph-io/badger/v3"
)

const (
	dbPath = "./tmp/blocks"
)

type BlockChain struct {
	LastHash []byte     // represents the last hash of the current block
	Database *badger.DB // represents the db where the blocks will be store
}

// InitBLockChain will start the blockchain
func InitBLockChain() *BlockChain {
	var lastHash []byte

	db, err := badger.Open(badger.DefaultOptions(dbPath))
	CheckError(err)

	err = db.Update(func(txn *badger.Txn) error {
		if _, err = txn.Get([]byte("lh")); err == badger.ErrKeyNotFound {
			fmt.Println("No existing blockchain found")
			genesis := genesis()
			fmt.Println("genesis proved")

			err = txn.Set(genesis.Hash, genesis.Serialize())
			CheckError(err)
			err = txn.Set([]byte("lh"), genesis.Hash)

			lastHash = genesis.Hash
			return err
		} else {
			item, err := txn.Get([]byte("lh"))
			CheckError(err)

			lastHash, err = item.ValueCopy(nil)
			return err
		}
	})

	CheckError(err)
	blockChain := &BlockChain{lastHash, db}
	return blockChain
}

// AddBlock will add a block to the block chain
func (chain *BlockChain) AddBlock(data string) {
	var lastHash []byte

	err := chain.Database.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte("lh"))
		CheckError(err)

		lastHash, err = item.ValueCopy(nil)
		return err
	})

	CheckError(err)
	newBlock := CreateBlock(data, lastHash)

	err = chain.Database.Update(func(txn *badger.Txn) error {
		err := txn.Set(newBlock.Hash, newBlock.Serialize())
		CheckError(err)

		err = txn.Set([]byte("lh"), newBlock.Hash)
		chain.LastHash = newBlock.Hash
		return err
	})

	CheckError(err)
}
