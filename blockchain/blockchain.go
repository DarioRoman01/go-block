package blockchain

import (
	"bytes"
	"crypto/ecdsa"
	"encoding/hex"
	"fmt"
	"os"
	"runtime"

	"github.com/dgraph-io/badger/v3"
	"github.com/pkg/errors"
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
func (chain *BlockChain) FindUnspentTransactions(pubKeyHash []byte) []Transaction {
	var unspentTsx []Transaction
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

				if out.IsLockedWithKey(pubKeyHash) {
					unspentTsx = append(unspentTsx, *tx)
				}
			}

			if !tx.IsCoinBase() {
				for _, in := range tx.Inputs {
					if in.UsesKey(pubKeyHash) {
						inTxID := hex.EncodeToString(in.ID)
						spentTxos[inTxID] = append(spentTxos[inTxID], in.Out)
					}
				}
			}
		}

		if len(block.PrevHash) == 0 {
			break // this is the genesis
		}
	}

	return unspentTsx
}

// Find unspent transaction outputs will return all the unspent
// outputs of the current user.
func (chain *BlockChain) FindUTXO(pubKeyHash []byte) []TxOutput {
	var Utxo []TxOutput
	unspentTransactoin := chain.FindUnspentTransactions(pubKeyHash)

	for _, tx := range unspentTransactoin {
		for _, out := range tx.Outputs {
			if out.IsLockedWithKey(pubKeyHash) {
				Utxo = append(Utxo, out)
			}
		}
	}

	return Utxo
}

// Find Spendable outputs will enable create normal transactions wich are not coinbase transactions
// this function will ensure that the user have the coins to make the transaction. something like
// the amount of coins that the user have
func (chain *BlockChain) FindSpendableOutputs(pubKeyHash []byte, amount int) (int, map[string][]int) {
	unspentOuts := make(map[string][]int)
	unspentTxs := chain.FindUnspentTransactions(pubKeyHash)
	accumulated := 0

Work:
	for _, tx := range unspentTxs {
		txID := hex.EncodeToString(tx.ID)

		for outIdx, out := range tx.Outputs {
			if out.IsLockedWithKey(pubKeyHash) && accumulated < amount {
				accumulated += out.Value
				unspentOuts[txID] = append(unspentOuts[txID], outIdx)

				if accumulated >= amount {
					break Work
				}
			}

		}
	}

	return accumulated, unspentOuts
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
	prevTxs := make(map[string]Transaction)
	for _, in := range tx.Inputs {
		prevTx, err := chain.FindTransaction(in.ID)
		CheckError(err)
		prevTxs[hex.EncodeToString(prevTx.ID)] = prevTx
	}

	return tx.Verify(prevTxs)
}

// has money 128arpUF8nrvzDKpkHhL7NrtGnHFZqNQkMtoNPgpxs8N6Nu5QqL
// no moneey 12TxQAbXQLP6266KzHgpiQJXKpbMCN4XJS37oXrAhWvEvwFtX6g
