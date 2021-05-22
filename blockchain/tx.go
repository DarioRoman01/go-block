package blockchain

import (
	"bytes"
	"encoding/gob"

	"github.com/Haizza1/go-block/wallet"
)

type TxOutput struct {
	Value      int    // represents the value in tokens
	PubKeyHash []byte // represents the public key
}

type TxOutputs struct {
	Outputs []TxOutput // represents the outputs in the list of outputs
}

type TxInput struct {
	ID        []byte // represents the transaction that the output is
	Out       int    // represents the index where the output appears
	Signature []byte // represents the data wich is use in the output pubkey
	PubKey    []byte // represents the public used in the transaction
}

// NewTXOuput will generate a new output instance
func NewTXOutput(value int, address string) *TxOutput {
	txo := &TxOutput{Value: value, PubKeyHash: nil}
	txo.Lock([]byte(address))

	return txo
}

// serialize will serialize the outputs struct into bytes
func (outs TxOutputs) Serialize() []byte {
	var buff bytes.Buffer
	encode := gob.NewEncoder(&buff)
	err := encode.Encode(outs)
	CheckError(err)
	return buff.Bytes()
}

// Deserialize will deserialize a chunk of bytes into a TxOutputs struct
func DeserializeOutputs(data []byte) TxOutputs {
	var outputs TxOutputs
	decode := gob.NewDecoder(bytes.NewReader(data))
	err := decode.Decode(&outputs)
	CheckError(err)
	return outputs
}

// UsesKey will check if the given publickey in equal
// to the transaction input pubkey hashed
func (in *TxInput) UsesKey(pubKeyHash []byte) bool {
	lockingHash := wallet.PublicKeyHash(in.PubKey)
	return bytes.Equal(lockingHash, pubKeyHash)
}

// Lock will set a hashed pubkey with base58 algorithm to the output
func (out *TxOutput) Lock(address []byte) {
	pubKeyHash := wallet.Base58Decode(address)
	pubKeyHash = pubKeyHash[1 : len(pubKeyHash)-4]
	out.PubKeyHash = pubKeyHash
}

// IsLocked with key will check is the given hash has been locked
func (out *TxOutput) IsLockedWithKey(pubKeyHash []byte) bool {
	return bytes.Equal(out.PubKeyHash, pubKeyHash)
}
