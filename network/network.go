package network

import (
	"bytes"
	"encoding/gob"
	"encoding/hex"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"os"
	"runtime"
	"syscall"

	"github.com/Haizza1/go-block/blockchain"
	"github.com/vrecan/death/v3"
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

// CmdToBytes will work like a serialize function for the
// commands recibed by the command line package
func CmdToBytes(cmd string) []byte {
	var bytes [commandLength]byte

	for i, v := range cmd {
		bytes[i] = byte(v)
	}

	return bytes[:]
}

// BytesToCmd will work like a deserialize function transforming
// the given bytes to a string
func BytesToCmd(bytes []byte) string {
	var cmd []byte

	for _, v := range bytes {
		if v != 0x0 {
			cmd = append(cmd, v)
		}
	}

	return fmt.Sprintf("%s", cmd)
}

// Request block will call send get block for each node in the list
func RequestBlocks() {
	for _, node := range KnownNodes {
		SendGetBlock(node)
	}
}

func ExtractCmd(request []byte) []byte {
	return request[:commandLength]
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
	// bestHeigth := chain.GetBestHeigth()
	payload := GobEncode(Version{Version: version, BestHeigth: 0, AddrFrom: nodeAddress})
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

	chain.AddBlock(block.Transactions)
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

	// blocks := chain.GetBlockHashes() TODO
	SendInv(payload.AddrFrom, "block", make([][]byte, 0))
}

// Handle GetData will handle the get data request
func HandleGetData(request []byte, chain *blockchain.BlockChain) {
	var payload GetData

	if err := DeserializePayload(request, &payload); err != nil {
		log.Panic(err)
	}

	if payload.Type == "block" {
		// TODO implement GetBlock
		// block, err := chain.GetBlock([]byte(payload.ID))
		// if err != nil {
		// 	return
		// }

		SendBlock(payload.AddrFrom, &blockchain.Block{})
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
	tx := blockchain.Transaction{ID: txData} //blockchain.DeserializeTx(txData)
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
	newBlock := blockchain.Block{} //chain.MineBlock(txs)

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

// handle version will handle the get version request
func HandleVersion(request []byte, chain *blockchain.BlockChain) {
	var payload Version
	if err := DeserializePayload(request, &payload); err != nil {
		log.Panic(err)
	}

	bestHeight := 0 //chain.GetBestHeigth() TODO
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

// GobEncode will allow us to serialize data in a generic way
func GobEncode(data interface{}) []byte {
	var buff bytes.Buffer

	encoder := gob.NewEncoder(&buff)
	if err := encoder.Encode(data); err != nil {
		log.Panic(err)
	}

	return buff.Bytes()
}

// Node is known will check if the given node is
// in the list of known nodes
func NodeIsKnown(addr string) bool {
	for _, node := range KnownNodes {
		if node == addr {
			return true
		}
	}

	return false
}

// Deserialze payload will deserialize the request into a struuct
func DeserializePayload(request []byte, payload interface{}) error {
	var buff bytes.Buffer
	buff.Write(request[commandLength:])
	dec := gob.NewDecoder(&buff)

	if err := dec.Decode(payload); err != nil {
		return err
	}
	return nil
}

// CloseDB will grafully shutdown the system if the process is interrupt or recive a syscall
func CloseDB(chain *blockchain.BlockChain) {
	d := death.NewDeath(syscall.SIGINT, syscall.SIGTERM, os.Interrupt)

	d.WaitForDeathWithFunc(func() {
		defer os.Exit(1)
		defer runtime.Goexit()
		chain.Database.Close()
	})
}
