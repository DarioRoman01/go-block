package network

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"net"

	"github.com/Haizza1/go-block/blockchain"
)

const (
	protocol      = "tcp"
	version       = 1
	commandLength = 12
)

var (
	nodeAddress     string
	minerAddress    string
	KnownNodes      = []string{"localhost:3000"}
	blocksInTransit = [][]byte{}
	memoryPool      = make(map[string]blockchain.Transaction)
)

type Addr struct {
	AddrList []string // represents the list of addresses of each of the nodes
}

type Block struct {
	AddrFrom string // represents the address that the block is build from
	Block    []byte // represents the block it self
}

type GetBlocks struct {
	AddrFrom string // represents the address where the block are being fetching
}

type GetData struct {
	AddrFrom string // represents the address where the block are being fetching
	Type     string // represents the type of the data that is being fetching
	ID       []byte // represents the id of the data
}

type Inv struct { // inventory struct
	AddrFrom string   // represents the address of the current node
	Type     string   // represents the type of data in the inventory
	Items    [][]byte // represents the items in the inventory
}

type Tx struct {
	AddrFrom    string // represents the address of the transaction
	Transaction []byte // represents the transaction
}

type Version struct {
	Version    int    // represents the version of the current blockchain node
	BestHeigth int    // represents the length of the blockchain
	AddrFrom   string // reprensents the address of the current node
}

// Request block will call send get block for each node in the list
func RequestBlocks() {
	for _, node := range KnownNodes {
		SendGetBlock(node)
	}
}

// send address will create the request of the address to be send
func SendAddr(address string) {
	nodes := Addr{AddrList: KnownNodes}
	nodes.AddrList = append(nodes.AddrList, nodeAddress)
	payload := GobEncode(nodes)
	request := append(CmdToBytes("addr"), payload...)

	sendData(address, request)
}

// Send block will create the request of the block to be send
func SendBlock(addr string, b *blockchain.Block) {
	data := Block{AddrFrom: nodeAddress, Block: b.Serialize()}
	payload := GobEncode(data)
	request := append(CmdToBytes("block"), payload...)

	sendData(addr, request)
}

// SendInv will the create request of the inventory to be send
func SendInv(addr, kind string, items [][]byte) {
	inventory := Inv{AddrFrom: nodeAddress, Type: kind, Items: items}
	payload := GobEncode(inventory)
	request := append(CmdToBytes("inv"), payload...)
	sendData(addr, request)
}

// SendTx will the create request of the transaction to be send
func SendTx(addr string, txn *blockchain.Transaction) {
	data := Tx{AddrFrom: nodeAddress, Transaction: txn.Serialize()}
	payload := GobEncode(data)
	request := append(CmdToBytes("tx"), payload...)
	sendData(addr, request)
}

// SendVersion will the create request of the Version to be send
func SendVersion(addr string, chain *blockchain.BlockChain) {
	bestHeigth := chain.GetBestHeigth()
	payload := GobEncode(Version{Version: version, BestHeigth: bestHeigth, AddrFrom: nodeAddress})
	request := append(CmdToBytes("version"), payload...)
	sendData(addr, request)
}

// SendGetBLock will the create request of the Getblock to be send
func SendGetBlock(addr string) {
	payload := GobEncode(GetBlocks{AddrFrom: nodeAddress})
	request := append(CmdToBytes("getblocks"), payload...)
	sendData(addr, request)
}

// SendGetData will the create request of the GetData to be send
func SendGetData(addr, kind string, id []byte) {
	payload := GobEncode(GetData{AddrFrom: nodeAddress, Type: kind, ID: id})
	request := append(CmdToBytes("getdata"), payload...)
	sendData(addr, request)
}

// SendData will send data from one node to another
func sendData(addr string, data []byte) {
	conn, err := net.Dial(protocol, addr)

	if err != nil {
		fmt.Printf("%s is not available\n", addr)
		var updatedNodes []string

		for _, node := range KnownNodes {
			if node != addr {
				updatedNodes = append(updatedNodes, node)
			}
		}

		KnownNodes = updatedNodes
		return
	}

	defer conn.Close()
	_, err = io.Copy(conn, bytes.NewReader(data))
	if err != nil {
		log.Panic(err)
	}
}

// mineTx will will nine the transaction
func MineTx(chain *blockchain.BlockChain) {
	var txs []*blockchain.Transaction

	for id := range memoryPool {
		fmt.Printf("tx: %s\n", memoryPool[id].ID)
		tx := memoryPool[id]
		if chain.VerifyTransaction(&tx) {
			txs = append(txs, &tx)
		}
	}

	if len(txs) == 0 {
		fmt.Println("All the transactions are invalid")
		return
	}

	cbTx := blockchain.CoinbaseTx(minerAddress, "")
	txs = append(txs, cbTx)
	newBlock := chain.MineBlock(txs)

	UtxoSet := blockchain.UTXOSet{BlockChain: chain}
	UtxoSet.Reindex()
	fmt.Println("New Block mined")

	for _, tx := range txs {
		txID := hex.EncodeToString(tx.ID)
		delete(memoryPool, txID)
	}

	for _, node := range KnownNodes {
		if node != nodeAddress {
			SendInv(node, "block", [][]byte{newBlock.Hash})
		}
	}

	if len(memoryPool) > 0 {
		MineTx(chain)
	}
}

// StartServer will start the server with the given node id
func StartServer(nodeID, mineAddress string) {
	nodeAddress = fmt.Sprintf("localhost:%s", nodeID)
	minerAddress = mineAddress
	ln, err := net.Listen(protocol, nodeAddress)
	if err != nil {
		log.Panic(err)
	}

	defer ln.Close()

	chain := blockchain.ContinueBlockChain(nodeID)
	defer chain.Database.Close()
	go CloseDB(chain)

	if nodeAddress != KnownNodes[0] {
		SendVersion(KnownNodes[0], chain)
	}

	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Panic(err)
		}

		go HandleConnection(conn, chain)
	}
}
