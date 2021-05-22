package blockchain

type TxOutput struct {
	Value  int    // represents the value in tokens
	PubKey string // represents the public key
}

type TxInput struct {
	ID  []byte // represents the transaction that the output is
	Out int    // represents the index where the output appears
	Sig string // represents the data wich is use in the output pubkey
}

// CanUnlock check if the given data is equal to the
// input sig wich is the pub key of the transaction
// if are equal means that the data of the input can be unlocked
func (in *TxInput) CanUnlock(data string) bool {
	return in.Sig == data
}

// CanBeUnlocked check if the given data is equal to the
// output pub key if are equal means that the data of the output can be unlocked
func (out *TxOutput) CanBeUnlocked(data string) bool {
	return out.PubKey == data
}
