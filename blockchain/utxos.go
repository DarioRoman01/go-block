package blockchain

import (
	"bytes"
	"encoding/hex"

	"github.com/dgraph-io/badger/v3"
)

var (
	// this variables helps to distinguish
	// the data atach to the utxos set and
	// the data atach to the blockcahin itself
	utxoPrefix = []byte("utxo-")
	// prefixLength = len(utxoPrefix)
)

// unspent transaction set
type UTXOSet struct {
	BlockChain *BlockChain // represents the blockchain
}

// Find unspent transaction outputs will return all the unspent
// outputs of the given public key
func (u UTXOSet) FindUTXO(pubKeyHash []byte) []TxOutput {
	var Utxo []TxOutput
	db := u.BlockChain.Database

	err := db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		it := txn.NewIterator(opts)
		defer it.Close()

		for it.Seek(utxoPrefix); it.ValidForPrefix(utxoPrefix); it.Next() {
			item := it.Item()
			v, err := item.ValueCopy(nil)
			CheckError(err)
			outs := DeserializeOutputs(v)

			for _, out := range outs.Outputs {
				if out.IsLockedWithKey(pubKeyHash) {
					Utxo = append(Utxo, out)
				}
			}
		}

		return nil
	})

	CheckError(err)
	return Utxo
}

// Find Spendable outputs will enable create normal transactions wich are not coinbase transactions
// this function will ensure that the user have the coins to make the transaction. something like
// the amount of coins that the user have
func (u UTXOSet) FindSpendableOutputs(pubKeyHash []byte, amount int) (int, map[string][]int) {
	unspentOuts := make(map[string][]int)
	accumulated := 0
	db := u.BlockChain.Database

	err := db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions

		it := txn.NewIterator(opts)
		defer it.Close()

		for it.Seek(utxoPrefix); it.ValidForPrefix(utxoPrefix); it.Next() {
			item := it.Item()
			key := item.Key()

			v, err := item.ValueCopy(nil)
			CheckError(err)

			k := bytes.TrimPrefix(key, utxoPrefix)
			txID := hex.EncodeToString(k)
			outs := DeserializeOutputs(v)

			for outIdx, out := range outs.Outputs {
				if out.IsLockedWithKey(pubKeyHash) && accumulated < amount {
					accumulated += out.Value
					unspentOuts[txID] = append(unspentOuts[txID], outIdx)
				}
			}
		}

		return nil
	})

	CheckError(err)
	return accumulated, unspentOuts
}

// count transactins will count all the transactions of
// unspent transactions outputs in the blockchain
func (u UTXOSet) CountTransactions() int {
	db := u.BlockChain.Database
	counter := 0

	err := db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		it := txn.NewIterator(opts)
		defer it.Close()

		for it.Seek(utxoPrefix); it.ValidForPrefix(utxoPrefix); it.Next() {
			counter++
		}

		return nil
	})

	CheckError(err)
	return counter
}

// update will update the unspent transactions set in the badger db
func (u *UTXOSet) Update(block *Block) {
	defer HandlePanic()

	db := u.BlockChain.Database
	err := db.Update(func(txn *badger.Txn) error {
		for _, tx := range block.Transactions {
			if !tx.IsCoinBase() {
				for _, in := range tx.Inputs {
					updateOuts := TxOutputs{}
					inID := append(utxoPrefix, in.ID...)

					item, err := txn.Get(inID)
					CheckError(err)

					v, err := item.ValueCopy(nil)
					CheckError(err)

					outs := DeserializeOutputs(v)

					for outIdx, out := range outs.Outputs {
						if outIdx != in.Out {
							updateOuts.Outputs = append(updateOuts.Outputs, out)
						}
					}

					if len(updateOuts.Outputs) == 0 {
						if err := txn.Delete(inID); err != nil {
							panic(err.Error())
						}
					} else {
						if err := txn.Set(inID, updateOuts.Serialize()); err != nil {
							panic(err.Error())
						}
					}
				}
			}

			newOutputs := TxOutputs{}
			newOutputs.Outputs = append(newOutputs.Outputs, tx.Outputs...)

			txID := append(utxoPrefix, tx.ID...)
			if err := txn.Set(txID, newOutputs.Serialize()); err != nil {
				panic(err.Error())
			}
		}
		return nil
	})
	CheckError(err)
}

// Reindex will delete all the data with the utxoprefix
// and the rebuild the set inside of the db
func (u UTXOSet) Reindex() {
	db := u.BlockChain.Database
	u.DeleteByPrefix(utxoPrefix)
	utxo := u.BlockChain.FindUnspentTransactions()

	err := db.Update(func(txn *badger.Txn) error {
		for txId, outs := range utxo {
			key, err := hex.DecodeString(txId)
			if err != nil {
				return err
			}

			key = append(utxoPrefix, key...)
			err = txn.Set(key, outs.Serialize())
			CheckError(err)
		}

		return badger.ErrNilCallback
	})

	CheckError(err)
}

// DeleteByPrefix will delete all the data in badger db with the given prefix
func (u *UTXOSet) DeleteByPrefix(prefix []byte) {

	// grafullt shutdown the system if we panic
	defer HandlePanic()

	// delete all given keys
	deleteKeys := func(keysForDelete [][]byte) error {
		if err := u.BlockChain.Database.Update(func(txn *badger.Txn) error {
			for _, key := range keysForDelete {
				if err := txn.Delete(key); err != nil {
					return err
				}
			}

			return nil
		}); err != nil {
			return err
		}

		return nil
	}

	// badger max collect size
	collectSize := 100000

	// read only function to get keys to be deleted
	u.BlockChain.Database.View(func(txn *badger.Txn) error {

		// set iterator optinos
		opts := badger.DefaultIteratorOptions
		opts.PrefetchValues = false
		it := txn.NewIterator(opts)
		defer it.Close()

		// collect all the keys that will be deleted
		keysForDelete := make([][]byte, 0, collectSize)

		// counter of the keys collected to ensure we dont pass badger max collect size
		keysCollected := 0

		// loop until we hit the max badger collect size
		// if we hit that maximum we delete all the keys collected
		// and restart the loop setting to 0 the keys for delete
		// and the keysCollected until we dont hit the max collectedsize
		for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
			key := it.Item().KeyCopy(nil)
			keysForDelete = append(keysForDelete, key)
			keysCollected++

			if keysCollected == collectSize {
				if err := deleteKeys(keysForDelete); err != nil {
					panic(err)
				}

				keysForDelete = make([][]byte, 0, collectSize)
				keysCollected = 0
			}
		}

		// remove all the keys that left for delete in the for loop
		if keysCollected > 0 {
			if err := deleteKeys(keysForDelete); err != nil {
				panic(err)
			}
		}

		return nil
	})
}
