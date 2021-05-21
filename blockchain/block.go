package blockchain

type BlockChain struct {
	Blocks []*Block // represents the blocks in the blockchain
}

type Block struct {
	Hash     []byte // represents the hash of the block
	Data     []byte // represents the data inside of the block
	PrevHash []byte // represents last block hash
	Nonce    int
}

// CreateBlock will generate a new Block instance with a pointer
func CreateBlock(data string, prevHash []byte) *Block {
	block := &Block{
		Hash:     []byte{},
		Data:     []byte(data),
		PrevHash: prevHash,
		Nonce:    0,
	}

	pow := NewProof(block)
	nonce, hash := pow.Run()
	block.Nonce = nonce
	block.Hash = hash[:]
	return block
}

// AddBlock will add a block to the block chain
func (chain *BlockChain) AddBlock(data string) {
	prevBlock := chain.Blocks[len(chain.Blocks)-1]
	new := CreateBlock(data, prevBlock.Hash)
	chain.Blocks = append(chain.Blocks, new)
}

// Genesis will create the first block in the blockchain
func genesis() *Block {
	return CreateBlock("genesis", []byte{})
}

// InitBLockChain will start the blockchain
func InitBLockChain() *BlockChain {
	return &BlockChain{[]*Block{genesis()}}
}
