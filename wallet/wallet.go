package wallet

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"runtime"
)

const (
	checksumLength = 4
	version        = byte(0x00)
)

// Wallet represents users Wallet in the blockchain
type Wallet struct {
	PrivateKey ecdsa.PrivateKey // represents the private key of the wallet
	PublicKey  []byte           // represents the plubic key of the wallet
}

// Address will generate the wallet address according to the
// specification wich a hash with checksum hash, the version hash,
// and the public key hash
func (w Wallet) Address() []byte {
	pubHash := PublicKeyHash(w.PublicKey)
	versionedHash := append([]byte{version}, pubHash...)
	checksum := Checksum(versionedHash)

	fullHash := append(versionedHash, checksum...)
	address := Base58Encode(fullHash)

	fmt.Printf("pub key: %x\n", w.PublicKey)
	fmt.Printf("pub hash: %x\n", pubHash)
	fmt.Printf("address: %x\n", address)

	return address
}

// NewKeyPair will create a new public and private key for the user
func NewKeyPair() (ecdsa.PrivateKey, []byte) {
	curve := elliptic.P256()

	private, err := ecdsa.GenerateKey(curve, rand.Reader)
	handle(err)

	pub := append(private.PublicKey.X.Bytes(), private.PublicKey.Y.Bytes()...)
	return *private, pub
}

// MakeWalllet will create a new wallet instance
func MakeWallet() *Wallet {
	private, public := NewKeyPair()
	return &Wallet{PrivateKey: private, PublicKey: public}
}

func PublicKeyHash(pubKey []byte) []byte {
	pubHash := sha256.Sum256(pubKey)
	hasher := sha256.New()
	_, err := hasher.Write(pubHash[:])
	handle(err)

	publicRipMD := hasher.Sum(nil)
	return publicRipMD
}

// checksum wiil hash two times the publickey
func Checksum(payload []byte) []byte {
	firstHash := sha256.Sum256(payload)
	secondHash := sha256.Sum256(firstHash[:])

	return secondHash[:checksumLength]
}

// handle will check if the error is not
// nil and gracefully shutdown the system
func handle(err error) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Println(err)
			runtime.Goexit()
		}
	}()

	if err != nil {
		panic(err)
	}
}
