package blockchain

import (
	"bytes"
	"crypto/sha256"
	"encoding/gob"
	"encoding/hex"
	"fmt"
	"log"
	"runtime"
)

type Transaction struct {
	ID      []byte     // represents the id of the transaction
	Inputs  []TxInput  // represents the inputs of the transaction
	Outputs []TxOutput // represents the outputs of the transaction
}

// setID will generate a hashed id for the transaction
func (tx *Transaction) setID() {
	var encoded bytes.Buffer
	var hash [32]byte

	encode := gob.NewEncoder(&encoded)
	err := encode.Encode(tx)
	CheckError(err)

	hash = sha256.Sum256(encoded.Bytes())
	tx.ID = hash[:]
}

// CoinbasTx will generate a new transaction instance with the given data
func CoinbaseTx(to, data string) *Transaction {
	if data == "" {
		data = fmt.Sprintf("Coins to %s", to)
	}

	txin := TxInput{ID: []byte{}, Out: -1, Sig: data}
	txout := TxOutput{Value: 100, PubKey: to}

	tx := Transaction{
		ID:      nil,
		Inputs:  []TxInput{txin},
		Outputs: []TxOutput{txout},
	}

	tx.setID()
	return &tx
}

// NewTransaction will create a new transacion and validate if the user has enough
// coins to make the transaction
func NewTransaction(from, to string, amount int, chain *BlockChain) *Transaction {
	defer func() {
		if r := recover(); r != nil {
			fmt.Println(r)
			runtime.Goexit()
		}
	}()

	var inputs []TxInput
	var outputs []TxOutput

	acc, validOutputs := chain.FindSpendableOutputs(from, amount)
	if acc < amount {
		log.Panic("Error: not enough funds")
	}

	for txid, outs := range validOutputs {
		txID, err := hex.DecodeString(txid)
		CheckError(err)

		for _, out := range outs {
			input := TxInput{ID: txID, Out: out, Sig: from}
			inputs = append(inputs, input)
		}
	}

	outputs = append(outputs, TxOutput{Value: amount, PubKey: to})
	if acc > amount {
		outputs = append(outputs, TxOutput{Value: acc - amount, PubKey: from})
	}

	tx := Transaction{ID: nil, Inputs: inputs, Outputs: outputs}
	tx.setID()
	return &tx
}

// IsCoinbase will determine if the current transaction is a coinbase
// based on the data created by default in the coinbase function
func (tx *Transaction) IsCoinBase() bool {
	return len(tx.Inputs) == 1 && len(tx.Inputs[0].ID) == 0 && tx.Inputs[0].Out == -1
}
