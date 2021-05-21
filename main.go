package main

import (
	"bytes"
	"crypto/sha256"
	"fmt"
)

type BlockChain struct {
	blocks []*Block // represents the blocks in the blockchain
}

type Block struct {
	Hash     []byte // represents the hash of the block
	Data     []byte // represents the data inside of the block
	PrevHash []byte // represents last block hash
}

// DeriveHash will generate Block current hash
func (b *Block) DeriveHash() {
	info := bytes.Join([][]byte{b.Data, b.PrevHash}, []byte{})
	hash := sha256.Sum256(info)
	b.Hash = hash[:]
}

// CreateBlock will generate a new Block instance with a pointer
func CreateBlock(data string, prevHash []byte) *Block {
	block := &Block{Hash: []byte{}, Data: []byte(data), PrevHash: prevHash}
	block.DeriveHash()
	return block
}

// AddBlock will add a block to the block chain
func (chain *BlockChain) AddBlock(data string) {
	prevBlock := chain.blocks[len(chain.blocks)-1]
	new := CreateBlock(data, prevBlock.Hash)
	chain.blocks = append(chain.blocks, new)
}

// Genesis will create the first block in the blockchain
func Genesis() *Block {
	return CreateBlock("genesis", []byte{})
}

// InitBLockChain will start the blockchain
func InitBLockChain() *BlockChain {
	return &BlockChain{[]*Block{Genesis()}}
}

func main() {
	chain := InitBLockChain()

	chain.AddBlock("First block after genesis")
	chain.AddBlock("Second block after genesis")
	chain.AddBlock("Third block after genesis")

	for _, block := range chain.blocks {
		fmt.Printf("Previos Hash: %x\n", block.PrevHash)
		fmt.Printf("Data in block: %s\n", block.Data)
		fmt.Printf("Block Hash: %x\n", block.Hash)
	}
}
