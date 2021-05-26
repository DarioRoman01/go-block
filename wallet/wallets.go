package wallet

import (
	"bytes"
	"crypto/elliptic"
	"encoding/gob"
	"fmt"
	"io/ioutil"
	"os"
)

const walletFile = "./tmp/Wallets_%s.data"

type Wallets struct {
	Wallets map[string]*Wallet
}

// CreateWallters will generate a new Wallets instance
func CreateWallets(nodeId string) (*Wallets, error) {
	wallets := &Wallets{}

	wallets.Wallets = make(map[string]*Wallet)
	err := wallets.LoadFile(nodeId)
	return wallets, err
}

// GetWallet will return the wallet associeted with the givenm
// address if exists.
func (ws *Wallets) GetWallet(address string) Wallet {
	return *ws.Wallets[address]
}

// GetAllAddress will return a slice of all addresses in the map
func (ws *Wallets) GetAllAddress() []string {
	var addresses []string

	for address := range ws.Wallets {
		addresses = append(addresses, address)
	}

	return addresses
}

// Add wallet will save a wallet into the map and the wallets file
func (ws *Wallets) AddWallet() string {
	wallet := MakeWallet()
	address := fmt.Sprintf("%s", wallet.Address())

	ws.Wallets[address] = wallet
	return address
}

// loadfile will check if the wallets file exists, if exists
// will decode the data into the wallets struct
func (ws *Wallets) LoadFile(nodeId string) error {
	walletFile := fmt.Sprintf(walletFile, nodeId)
	if _, err := os.Stat(walletFile); os.IsNotExist(err) {
		return err
	}

	var wallets Wallets
	fileContent, err := ioutil.ReadFile(walletFile)
	if err != nil {
		return err
	}

	gob.Register(elliptic.P256())
	decoder := gob.NewDecoder(bytes.NewReader(fileContent))
	err = decoder.Decode(&wallets)
	if err != nil {
		return err
	}

	ws.Wallets = wallets.Wallets
	return nil
}

// save file will save all data in the wallets map into the
// wallets.data file
func (ws *Wallets) SaveFile(nodeId string) {
	var content bytes.Buffer
	walletFile := fmt.Sprintf(walletFile, nodeId)
	gob.Register(elliptic.P256())

	encoder := gob.NewEncoder(&content)
	err := encoder.Encode(ws)
	handle(err)

	err = ioutil.WriteFile(walletFile, content.Bytes(), 0644)
	handle(err)
}
