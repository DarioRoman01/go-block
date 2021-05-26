package blockchain

import (
	"bytes"
	"crypto/ecdsa"
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"runtime"

	"github.com/dgraph-io/badger/v3"
	"github.com/pkg/errors"
)

const (
	dbPath      = "./tmp/blocks_%s"
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
func DBexists(path string) bool {
	if _, err := os.Stat(path + "/MANIFEST"); os.IsNotExist(err) {
		return false
	}

	return true
}

// ContinueBlockchain will continue the blockchain with the last hashed block
func ContinueBlockChain(nodeId string) *BlockChain {
	path := fmt.Sprintf(dbPath, nodeId)
	if !DBexists(path) {
		fmt.Println("No existing blockchain found, go and create one!")
		runtime.Goexit()
	}

	var lastHash []byte
	db, err := badger.Open(badger.DefaultOptions(path))
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
func InitBLockChain(address, nodeId string) *BlockChain {
	var lastHash []byte
	path := fmt.Sprintf(dbPath, nodeId)

	if DBexists(path) {
		fmt.Println("Blockchain already exists")
		runtime.Goexit()
	}

	db, err := badger.Open(badger.DefaultOptions(path))
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

// add block will add the block to the db and check if
// the heigth is the grater in the blockchain
func (chain *BlockChain) AddBlock(block *Block) {
	err := chain.Database.Update(func(txn *badger.Txn) error {
		if _, err := txn.Get(block.Hash); err != nil {
			return nil // the block is already store
		}

		blockData := block.Serialize()
		if err := txn.Set(block.Hash, blockData); err != nil {
			return err
		}

		item, err := txn.Get([]byte("lh"))
		if err != nil {
			return err
		}

		lastHash, _ := item.ValueCopy(nil)

		item, err = txn.Get(lastHash)
		if err != nil {
			return err
		}

		lastBlockData, _ := item.ValueCopy(nil)
		lastBlock := Deserialize(lastBlockData)

		if block.Heigth > lastBlock.Heigth {
			err = txn.Set([]byte("lh"), block.Hash)
			if err != nil {
				return err
			}

			chain.LastHash = block.Hash
		}

		return nil
	})

	CheckError(err)
}

// Get block will retrieve the block from the db if exists
func (chain *BlockChain) GetBlock(blockHash []byte) (Block, error) {
	var block Block

	err := chain.Database.View(func(txn *badger.Txn) error {
		if item, err := txn.Get(blockHash); err != nil {
			return errors.New("Block is not found")
		} else {
			blockData, err := item.ValueCopy(nil)
			if err != nil {
				return err
			}

			block = *Deserialize(blockData)
		}
		return nil
	})

	if err != nil {
		return block, err
	}

	return block, nil
}

// Get block hashes will retrieve a 2 dimensional array
// of all block hashes in the blockchain
func (chain *BlockChain) GetBlockHashes() [][]byte {
	var blocks [][]byte
	iter := chain.Iterator()

	for {
		block := iter.Next()
		blocks = append(blocks, block.Hash)
		if len(block.PrevHash) == 0 {
			break
		}
	}

	return blocks
}

// Get BestHeigth will retrieve the larger block heigth
func (chain *BlockChain) GetBestHeigth() int {
	var lastBlock Block

	err := chain.Database.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte("lh"))
		if err != nil {
			return err
		}
		lasthash, _ := item.ValueCopy(nil)

		item, err = txn.Get(lasthash)
		if err != nil {
			return err
		}

		lastBlockData, _ := item.ValueCopy(nil)
		lastBlock = *Deserialize(lastBlockData)
		return nil
	})

	CheckError(err)
	return lastBlock.Heigth
}

// MineBlock will add a block to the block chain
func (chain *BlockChain) MineBlock(transactions []*Transaction) *Block {
	var lastHash []byte
	var lastHeigth int

	defer HandlePanic()
	for _, tx := range transactions {
		if !chain.VerifyTransaction(tx) {
			log.Panic("Invalid transaction")
		}
	}

	err := chain.Database.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte("lh"))
		if err != nil {
			return err
		}

		lastHash, err = item.ValueCopy(nil)
		if err != nil {
			return err
		}

		item, err = txn.Get(lastHash)
		if err != nil {
			return err
		}

		lastBlockData, err := item.ValueCopy(nil)
		lastBlock := Deserialize(lastBlockData)
		lastHeigth = lastBlock.Heigth

		return err
	})

	CheckError(err)
	newBlock := CreateBlock(transactions, lastHash, lastHeigth+1)

	err = chain.Database.Update(func(txn *badger.Txn) error {
		err := txn.Set(newBlock.Hash, newBlock.Serialize())
		if err != nil {
			return err
		}

		err = txn.Set([]byte("lh"), newBlock.Hash)
		chain.LastHash = newBlock.Hash
		return err
	})

	CheckError(err)
	return newBlock
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
		if err != nil {
			return err
		}

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
func (chain *BlockChain) FindUnspentTransactions() map[string]TxOutputs {
	unspentTxos := make(map[string]TxOutputs)
	spentTxos := make(map[string][]int)
	iter := chain.Iterator()

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

				outs := unspentTxos[txID]
				outs.Outputs = append(outs.Outputs, out)
				unspentTxos[txID] = outs

			}

			if !tx.IsCoinBase() {
				for _, in := range tx.Inputs {
					inTxID := hex.EncodeToString(in.ID)
					spentTxos[inTxID] = append(spentTxos[inTxID], in.Out)
				}
			}
		}

		if len(block.PrevHash) == 0 {
			break // this is the genesis
		}
	}

	return unspentTxos
}

// FindTransaction will check in the blockchain if the given transaction ID exists
// if exits its return else we return a error
func (chain *BlockChain) FindTransaction(ID []byte) (Transaction, error) {
	iter := chain.Iterator()

	for {
		block := iter.Next()

		for _, tx := range block.Transactions {
			if bytes.Equal(tx.ID, ID) {
				return *tx, nil
			}
		}

		if len(block.PrevHash) == 0 {
			break // this is the genesis
		}
	}

	return Transaction{}, errors.New("Transaction does not exists")
}

// SignTransaction will sign the transaction with the user private key
func (chain *BlockChain) SingTransaction(tx *Transaction, privKey ecdsa.PrivateKey) {
	prevTxs := make(map[string]Transaction)

	for _, in := range tx.Inputs {
		prevTx, err := chain.FindTransaction(in.ID)
		CheckError(err)
		prevTxs[hex.EncodeToString(prevTx.ID)] = prevTx
	}

	tx.Sign(privKey, prevTxs)
}

// VerifyTransaction will check if the given transaction is valid
func (chain *BlockChain) VerifyTransaction(tx *Transaction) bool {
	if tx.IsCoinBase() {
		return true
	}

	prevTxs := make(map[string]Transaction)
	for _, in := range tx.Inputs {
		prevTx, err := chain.FindTransaction(in.ID)
		CheckError(err)
		prevTxs[hex.EncodeToString(prevTx.ID)] = prevTx
	}

	return tx.Verify(prevTxs)
}
