package main

import (
	"fmt"

	"github.com/Haizza1/go-block/blockchain"
)

func main() {

	chain := blockchain.InitBLockChain()

	chain.AddBlock("First block after genesis")
	chain.AddBlock("Second block after genesis")
	chain.AddBlock("Third block after genesis")

	for _, block := range chain.Blocks {
		fmt.Printf("Previos Hash: %x\n", block.PrevHash)
		fmt.Printf("Data in block: %s\n", block.Data)
		fmt.Printf("Block Hash: %x\n", block.Hash)
	}
}
