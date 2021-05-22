package blockchain

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/gob"
	"encoding/hex"
	"fmt"
	"log"
	"math/big"
	"runtime"
	"strings"
	"sync"

	"github.com/Haizza1/go-block/wallet"
)

type Transaction struct {
	ID      []byte     // represents the id of the transaction
	Inputs  []TxInput  // represents the inputs of the transaction
	Outputs []TxOutput // represents the outputs of the transaction
}

// Serialze will serialize the transaction struct into bytes
func (tx *Transaction) Serialize() []byte {
	var encoded bytes.Buffer

	enc := gob.NewEncoder(&encoded)
	err := enc.Encode(tx)
	CheckError(err)

	return encoded.Bytes()
}

// Hash will create a new hash with the transaction data
func (tx *Transaction) Hash() []byte {
	var hash [32]byte

	txCopy := *tx
	txCopy.ID = []byte{}
	hash = sha256.Sum256(txCopy.Serialize())
	return hash[:]
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

	txin := TxInput{ID: []byte{}, Out: -1, Signature: []byte(data)}
	txout := NewTXOutput(100, to)

	tx := Transaction{
		ID:      nil,
		Inputs:  []TxInput{txin},
		Outputs: []TxOutput{*txout},
	}

	tx.setID()
	return &tx
}

// NewTransaction will create a new transacion and validate if the user has enough
// coins to make the transaction
func NewTransaction(from, to string, amount int, chain *BlockChain) *Transaction {
	defer HandlePanic()
	var inputs []TxInput
	var outputs []TxOutput

	wallets, err := wallet.CreateWallets()
	CheckError(err)
	w := wallets.GetWallet(from)
	pubKeyHash := wallet.PublicKeyHash(w.PublicKey)

	acc, validOutputs := chain.FindSpendableOutputs(pubKeyHash, amount)
	if acc < amount {
		log.Panic("Error: not enough funds")
	}

	for txid, outs := range validOutputs {
		txID, err := hex.DecodeString(txid)
		CheckError(err)

		for _, out := range outs {
			input := TxInput{ID: txID, Out: out, Signature: nil, PubKey: w.PublicKey}
			inputs = append(inputs, input)
		}
	}

	outputs = append(outputs, *NewTXOutput(amount, to))
	if acc > amount {
		outputs = append(outputs, *NewTXOutput(acc-amount, from))
	}

	tx := Transaction{ID: nil, Inputs: inputs, Outputs: outputs}
	tx.ID = tx.Hash()
	chain.SingTransaction(&tx, w.PrivateKey)
	return &tx
}

// IsCoinbase will determine if the current transaction is a coinbase
// based on the data created by default in the coinbase function
func (tx *Transaction) IsCoinBase() bool {
	return len(tx.Inputs) == 1 && len(tx.Inputs[0].ID) == 0 && tx.Inputs[0].Out == -1
}

// Sign will allow to sign and verify the transactions
func (tx *Transaction) Sign(privKey ecdsa.PrivateKey, prexTxs map[string]Transaction) {
	if tx.IsCoinBase() {
		return
	}

	defer HandlePanic()
	for _, in := range tx.Inputs {
		if prexTxs[hex.EncodeToString(in.ID)].ID == nil {
			panic("ERROR: Previos transaction is not correct!")
		}
	}

	txCopy := tx.TrimmedCopy()

	for inId, in := range txCopy.Inputs {
		prevTx := prexTxs[hex.EncodeToString(in.ID)]
		txCopy.Inputs[inId].Signature = nil
		txCopy.Inputs[inId].PubKey = prevTx.Outputs[in.Out].PubKeyHash
		txCopy.ID = txCopy.Hash()
		txCopy.Inputs[inId].PubKey = nil

		r, s, err := ecdsa.Sign(rand.Reader, &privKey, txCopy.ID)
		CheckError(err)
		signature := append(r.Bytes(), s.Bytes()...)
		tx.Inputs[inId].Signature = signature
	}
}

// trimmed copy will return a copy of the given transaction
func (tx *Transaction) TrimmedCopy() Transaction {
	var inputs []TxInput
	var outputs []TxOutput

	wg := sync.WaitGroup{}
	wg.Add(2)

	go func() {
		for _, in := range tx.Inputs {
			inputs = append(inputs, TxInput{
				ID:        in.ID,
				Out:       in.Out,
				Signature: nil,
				PubKey:    nil,
			})
		}

		wg.Done()
	}()

	go func() {
		for _, out := range tx.Outputs {
			outputs = append(outputs, TxOutput{Value: out.Value, PubKeyHash: out.PubKeyHash})
		}

		wg.Done()
	}()

	wg.Wait()
	txCopy := Transaction{ID: tx.ID, Inputs: inputs, Outputs: outputs}
	return txCopy
}

// Verify will check that the current transaction is valid
func (tx *Transaction) Verify(prevTxs map[string]Transaction) bool {
	if tx.IsCoinBase() {
		return true
	}

	defer HandlePanic()
	for _, in := range tx.Inputs {
		if prevTxs[hex.EncodeToString(in.ID)].ID == nil {
			panic("Previos transaction does not exists :(")
		}
	}

	txCopy := tx.TrimmedCopy()
	curve := elliptic.P256()

	for inId, in := range tx.Inputs {
		prevTx := prevTxs[hex.EncodeToString(in.ID)]
		txCopy.Inputs[inId].Signature = nil
		txCopy.Inputs[inId].PubKey = prevTx.Outputs[in.Out].PubKeyHash
		txCopy.ID = txCopy.Hash()
		txCopy.Inputs[inId].PubKey = nil

		r := big.Int{}
		s := big.Int{}
		sigLen := len(in.Signature)
		r.SetBytes(in.Signature[:(sigLen / 2)])
		s.SetBytes(in.Signature[(sigLen / 2):])

		x := big.Int{}
		y := big.Int{}
		keyLen := len(in.PubKey)
		x.SetBytes(in.PubKey[:(keyLen / 2)])
		y.SetBytes(in.PubKey[(keyLen / 2):])

		rawPubKey := ecdsa.PublicKey{Curve: curve, X: &x, Y: &y}
		if !ecdsa.Verify(&rawPubKey, txCopy.ID, &r, &s) {
			return false
		}
	}

	return true
}

// String will return a string representation of the transaction
func (tx Transaction) String() string {
	var lines []string

	lines = append(lines, fmt.Sprintf("--- Transaction %x:", tx.ID))
	for i, input := range tx.Inputs {
		lines = append(lines, fmt.Sprintf("     Input %d:", i))
		lines = append(lines, fmt.Sprintf("       TXID:     %x", input.ID))
		lines = append(lines, fmt.Sprintf("       Out:       %d", input.Out))
		lines = append(lines, fmt.Sprintf("       Signature: %x", input.Signature))
		lines = append(lines, fmt.Sprintf("       PubKey:    %x", input.PubKey))
	}

	for i, output := range tx.Outputs {
		lines = append(lines, fmt.Sprintf("     Output %d:", i))
		lines = append(lines, fmt.Sprintf("       Value:  %d", output.Value))
		lines = append(lines, fmt.Sprintf("       Script: %x", output.PubKeyHash))
	}

	return strings.Join(lines, "\n")
}

// handle panic will recover from panics and then
// gracefully shutdown the system. This necesary
// because badger needs time to collect all the
// garbage in the system
func HandlePanic() {
	if r := recover(); r != nil {
		fmt.Println(r)
		runtime.Goexit()
	}
}
