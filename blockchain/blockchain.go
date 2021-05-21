package blockchain

import (
	"fmt"
	"os"
	"runtime"

	"github.com/dgraph-io/badger/v3"
)

const (
	dbPath      = "./tmp/blocks"
	dbFile      = "./tmp/blocks/MANIFEST"
	genesisData = "First Transaction from genesis"
)

type BlockChain struct {
	LastHash []byte     // represents the last hash of the current block
	Database *badger.DB // represents the db where the blocks will be store
}

type BlockChainIterator struct {
	CurrentHash []byte // represents the current hash
	Database    *badger.DB
}

//DBexists will check if a badger db already exists in the db path
func DBexists() bool {
	if _, err := os.Stat(dbFile); os.IsNotExist(err) {
		return false
	}

	return true
}

// ContinueBlockchain will continue the blockchain with the last hashed block
func ContinueBlockChain(address string) *BlockChain {
	if !DBexists() {
		fmt.Println("No existing blockchain found, go and create one!")
		runtime.Goexit()
	}

	var lastHash []byte
	db, err := badger.Open(badger.DefaultOptions(dbPath))
	CheckError(err)

	err = db.Update(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte("lh"))
		CheckError(err)

		lastHash, err = item.ValueCopy(nil)
		return err
	})

	CheckError(err)
	blockChain := &BlockChain{lastHash, db}
	return blockChain
}

// InitBLockChain will start the blockchain
func InitBLockChain(address string) *BlockChain {
	var lastHash []byte

	if DBexists() {
		fmt.Println("Blockchain already exists")
		runtime.Goexit()
	}

	db, err := badger.Open(badger.DefaultOptions(dbPath))
	CheckError(err)

	err = db.Update(func(txn *badger.Txn) error {
		cbtx := CoinbaseTx(address, genesisData)
		genesis := Genesis(cbtx)
		fmt.Println("Genesis Created")

		err := txn.Set(genesis.Hash, genesis.Serialize())
		CheckError(err)

		err = txn.Set([]byte("lh"), genesis.Hash)
		lastHash = genesis.Hash
		return err
	})

	CheckError(err)
	blockChain := &BlockChain{lastHash, db}
	return blockChain
}

// AddBlock will add a block to the block chain
func (chain *BlockChain) AddBlock(transactions []*Transaction) {
	var lastHash []byte

	err := chain.Database.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte("lh"))
		CheckError(err)

		lastHash, err = item.ValueCopy(nil)
		return err
	})

	CheckError(err)
	newBlock := CreateBlock(transactions, lastHash)

	err = chain.Database.Update(func(txn *badger.Txn) error {
		err := txn.Set(newBlock.Hash, newBlock.Serialize())
		CheckError(err)

		err = txn.Set([]byte("lh"), newBlock.Hash)
		chain.LastHash = newBlock.Hash
		return err
	})

	CheckError(err)
}

// Iterator will return a new block chain iterator instance
// whit the blockchain data
func (chain *BlockChain) Iterator() *BlockChainIterator {
	iter := &BlockChainIterator{chain.LastHash, chain.Database}
	return iter
}

// Next will return the next block on the list, until the genesis
func (iter *BlockChainIterator) Next() *Block {
	var block *Block

	err := iter.Database.View(func(txn *badger.Txn) error {
		item, err := txn.Get(iter.CurrentHash)
		CheckError(err)

		endcodedBlock, err := item.ValueCopy(nil)
		block = Deserialize(endcodedBlock)
		return err
	})

	CheckError(err)

	iter.CurrentHash = block.PrevHash
	return block
}

// FindUnspentTransactions will find all unspent transactions assing to one address.
// Unspent transactions are transactions that have outputs wich are not referenced
// by other inputs. This is important because if there is an output hassent been spent
// that means that those tokens still exists for a certain user.
// func (chain *BlockChain) FindUnspentTransactions(address string) []Transaction {
// 	var unspentTsx []Transaction
// 	spentTxos := make(map[string][]int)
// 	iter := chain.Iterator()

// 	for {
// 		block := iter.Next()

// 		if len(block.PrevHash) == 0 {
// 			break // this is the genesis
// 		}
// 	}

// 	return unspentTsx
// }
