package network

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"log"
	"net"

	"github.com/Haizza1/go-block/blockchain"
)

// handle version will handle the get version request
func HandleVersion(request []byte, chain *blockchain.BlockChain) {
	var payload Version
	if err := DeserializePayload(request, &payload); err != nil {
		log.Panic(err)
	}

	bestHeight := chain.GetBestHeigth()
	otherHeigth := payload.BestHeigth

	if bestHeight < otherHeigth {
		SendGetBlock(payload.AddrFrom)

	} else if bestHeight > otherHeigth {
		SendVersion(payload.AddrFrom, chain)
	}

	if !NodeIsKnown(payload.AddrFrom) {
		KnownNodes = append(KnownNodes, payload.AddrFrom)
	}
}

// handle address will handle the address get address request
func HandleAddr(request []byte) {
	var payload Addr
	if err := DeserializePayload(request, &payload); err != nil {
		log.Panic(err)
	}

	KnownNodes = append(KnownNodes, payload.AddrList...)
	fmt.Printf("there are %d, known nodes\n", len(KnownNodes))
	RequestBlocks()
}

// handle inventory will handle the get Inv request
func HandleInv(request []byte, chain *blockchain.BlockChain) {
	var payload Inv

	if err := DeserializePayload(request, &payload); err != nil {
		log.Panic(err)
	}

	fmt.Printf("Recivied inventory with %d, %s\n", len(payload.Items), payload.Type)

	if payload.Type == "block" {
		blocksInTransit = payload.Items
		blockHash := payload.Items[0]
		SendGetData(payload.AddrFrom, "block", blockHash)

		newInTransit := [][]byte{}

		for _, v := range blocksInTransit {
			if !bytes.Equal(v, blockHash) {
				newInTransit = append(newInTransit, v)
			}
		}

		blocksInTransit = newInTransit
	}

	if payload.Type == "tx" {
		txID := payload.Items[0]
		if memoryPool[hex.EncodeToString(txID)].ID == nil {
			SendGetData(payload.AddrFrom, "tx", txID)
		}
	}
}

// handle block will handle the address get block request
func HandeBlock(request []byte, chain *blockchain.BlockChain) {
	var payload Block
	if err := DeserializePayload(request, &payload); err != nil {
		log.Panic(err)
	}

	blockData := payload.Block
	block := blockchain.Deserialize(blockData)
	fmt.Println("Recevied a new block!")

	chain.MineBlock(block.Transactions)
	fmt.Printf("Added block %x\n", block.Hash)

	if len(blocksInTransit) > 0 {
		blockHash := blocksInTransit[0]
		SendGetData(payload.AddrFrom, "block", blockHash)

		blocksInTransit = blocksInTransit[1:]
	} else {
		UtxoSet := blockchain.UTXOSet{BlockChain: chain}
		UtxoSet.Reindex()
	}
}

// handle get block will handle the get blocks request
func HandleGetBlocks(request []byte, chain *blockchain.BlockChain) {
	var payload GetBlocks
	if err := DeserializePayload(request, &payload); err != nil {
		log.Panic(err)
	}

	blocks := chain.GetBlockHashes()
	SendInv(payload.AddrFrom, "block", blocks)
}

// Handle GetData will handle the get data request
func HandleGetData(request []byte, chain *blockchain.BlockChain) {
	var payload GetData

	if err := DeserializePayload(request, &payload); err != nil {
		log.Panic(err)
	}

	if payload.Type == "block" {
		block, err := chain.GetBlock([]byte(payload.ID))
		if err != nil {
			return
		}

		SendBlock(payload.AddrFrom, &block)
	}

	if payload.Type == "tx" {
		txID := hex.EncodeToString(payload.ID)
		tx := memoryPool[txID]
		SendTx(payload.AddrFrom, &tx)
	}
}

// Handle transaction will handle the get transaction request
func HandleTx(request []byte, chain *blockchain.BlockChain) {
	var payload Tx
	if err := DeserializePayload(request, &payload); err != nil {
		log.Panic(err)
	}

	txData := payload.Transaction
	tx := blockchain.DeserializeTransaction(txData)
	memoryPool[hex.EncodeToString(tx.ID)] = tx

	fmt.Printf("%s, %d", nodeAddress, len(memoryPool))

	if nodeAddress == KnownNodes[0] {
		for _, node := range KnownNodes {
			if node != nodeAddress && node != payload.AddrFrom {
				SendInv(node, "tx", [][]byte{tx.ID})
			}
		}
	} else {
		if len(memoryPool) >= 2 && len(minerAddress) > 0 {
			MineTx(chain)
		}
	}
}

// HandleConnection will handle connections of the current node
func HandleConnection(conn net.Conn, chain *blockchain.BlockChain) {
	req, err := ioutil.ReadAll(conn)
	defer conn.Close()

	if err != nil {
		log.Panic(err)
	}

	command := BytesToCmd(req[:commandLength])
	fmt.Printf("Received %s command\n", command)

	switch command {
	case "addr":
		HandleAddr(req)

	case "block":
		HandeBlock(req, chain)

	case "inv":
		HandleInv(req, chain)

	case "getblcoks":
		HandleGetBlocks(req, chain)

	case "getdata":
		HandleGetData(req, chain)

	case "tx":
		HandleTx(req, chain)

	case "version":
		HandleVersion(req, chain)

	default:
		fmt.Println("Unkown Command")
	}
}
