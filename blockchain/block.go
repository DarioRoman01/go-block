package blockchain

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"log"
	"runtime"
)

type Block struct {
	Hash         []byte         // represents the hash of the block
	Transactions []*Transaction // represents the transactions of the block
	PrevHash     []byte         // represents last block hash
	Nonce        int            // represents the diffycul
}

// HashTransactions will allow to use a hashing mechanism
// to provide a unique reperesentation of all the transactions
func (b *Block) HashTransactions() []byte {
	var tsxHashes [][]byte

	for _, tx := range b.Transactions {
		tsxHashes = append(tsxHashes, tx.Serialize())
	}

	tree := NewMerkletree(tsxHashes)
	return tree.RootNode.Data
}

// CreateBlock will generate a new Block instance with a pointer
func CreateBlock(tsx []*Transaction, prevHash []byte) *Block {
	block := &Block{
		Hash:         []byte{},
		Transactions: tsx,
		PrevHash:     prevHash,
		Nonce:        0,
	}

	pow := NewProof(block)
	nonce, hash := pow.Run()
	block.Nonce = nonce
	block.Hash = hash[:]
	return block
}

// Genesis will create the first block in the blockchain
func Genesis(coinbase *Transaction) *Block {
	return CreateBlock([]*Transaction{coinbase}, []byte{})
}

// Serialize will serializer the block struct in to bytes
func (b *Block) Serialize() []byte {
	var res bytes.Buffer
	encoder := gob.NewEncoder(&res)

	err := encoder.Encode(b)
	CheckError(err)
	return res.Bytes()
}

// Deserialize will deserialize a chunk of data into a Block struct
func Deserialize(data []byte) *Block {
	var block Block

	decoder := gob.NewDecoder(bytes.NewReader(data))
	err := decoder.Decode(&block)
	CheckError(err)

	return &block
}

// CheckError will check if there is any error and then gracefuly shutdown the system
func CheckError(err error) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("Error: ", r)
			runtime.Goexit()
		}
	}()

	if err != nil {
		log.Panic(err)
	}
}
