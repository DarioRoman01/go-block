package wallet

import "github.com/mr-tron/base58"

// Base58Encode will use base58 algorithm to encode the input
func Base58Encode(input []byte) []byte {
	encode := base58.Encode(input)

	return []byte(encode)
}

// Base58Decode will  use base58 algorithm to decode the input
func Base58Decode(input []byte) []byte {
	decode, err := base58.Decode(string(input[:]))
	handle(err)
	return decode
}
