package blockchain

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"log"
	"math"
	"math/big"
)

/*
proof of work algorithm
steps:
	1. take the data from the block
	2. create a counter (nonce) wich starts at 0
	3. create a hash of the data plus the counter
	4. check the hash of the see if it meets a set of requirements

requirements:
	The fist few bytes must constains 0s
*/

const difficulty = 12

type ProofOfWork struct {
	Block  *Block
	Target *big.Int
}

// NewProof will create a new Proof of work instance
func NewProof(b *Block) *ProofOfWork {
	target := big.NewInt(1)
	target.Lsh(target, uint(256-difficulty))
	pow := &ProofOfWork{b, target}
	return pow
}

// IinitData will create a new byte slice and return it
func (pow *ProofOfWork) InitData(nonce int) []byte {
	data := bytes.Join(
		[][]byte{
			pow.Block.PrevHash,
			pow.Block.Data,
			ToHex(int64(nonce)),
			ToHex(int64(difficulty)),
		},
		[]byte{},
	)

	return data
}

// Run will create a hash from the data + countter
// and then check if the hash meet a set of requirements
func (pow *ProofOfWork) Run() (int, []byte) {
	var initHash big.Int
	var hash [32]byte
	nonce := 0

	for nonce < math.MaxInt64 {
		data := pow.InitData(nonce)
		hash = sha256.Sum256(data)
		fmt.Printf("\r%x", hash)

		initHash.SetBytes(hash[:])

		if initHash.Cmp(pow.Target) == -1 {
			break
		} else {
			nonce++
		}
	}

	fmt.Println()
	return nonce, hash[:]
}

// Tohex will decode the given number into bytes, set it in
// the bytes buffer and return the bytes porcion of the buffer
func ToHex(num int64) []byte {
	buff := new(bytes.Buffer)
	err := binary.Write(buff, binary.BigEndian, num)
	if err != nil {
		log.Panic(err)
	}

	return buff.Bytes()
}
