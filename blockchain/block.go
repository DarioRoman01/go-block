package blockchain

import (
	"bytes"
	"encoding/gob"
	"log"
)

type Block struct {
	Hash     []byte // represents the hash of the block
	Data     []byte // represents the data inside of the block
	PrevHash []byte // represents last block hash
	Nonce    int
}

// CreateBlock will generate a new Block instance with a pointer
func CreateBlock(data string, prevHash []byte) *Block {
	block := &Block{
		Hash:     []byte{},
		Data:     []byte(data),
		PrevHash: prevHash,
		Nonce:    0,
	}

	pow := NewProof(block)
	nonce, hash := pow.Run()
	block.Nonce = nonce
	block.Hash = hash[:]
	return block
}

// Genesis will create the first block in the blockchain
func genesis() *Block {
	return CreateBlock("genesis", []byte{})
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

func CheckError(err error) {
	if err != nil {
		log.Panic(err)
	}
}
