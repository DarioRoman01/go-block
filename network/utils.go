package network

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"log"
	"os"
	"runtime"
	"syscall"

	"github.com/Haizza1/go-block/blockchain"
	"github.com/vrecan/death/v3"
)

func ExtractCmd(request []byte) []byte {
	return request[:commandLength]
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
