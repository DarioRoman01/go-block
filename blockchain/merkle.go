package blockchain

import "crypto/sha256"

type MerkleTree struct {
	RootNode *MerkleNode // represents the Root of the merkle tree
}

type MerkleNode struct {
	Left  *MerkleNode // represents the left child of the node
	Rigth *MerkleNode // represents the rigth child of the node
	Data  []byte      // represents the data in the node
}

// NewMerkleNode will generate a new merkle tree node instance
func NewMerkleNode(left, rigth *MerkleNode, data []byte) *MerkleNode {
	node := &MerkleNode{}

	if left == nil && rigth == nil {
		hash := sha256.Sum256(data)
		node.Data = hash[:]
	} else {
		prevHashes := append(left.Data, rigth.Data...)
		hash := sha256.Sum256(prevHashes)
		node.Data = hash[:]
	}

	node.Left = left
	node.Rigth = rigth.Left
	return node
}

// NewMerkleTree will generate a new merkle tree instance
func NewMerkletree(data [][]byte) *MerkleTree {
	var nodes []MerkleNode

	if len(data)%2 != 0 {
		data = append(data, data[len(data)-1])
	}

	for _, dat := range data {
		node := NewMerkleNode(nil, nil, dat)
		nodes = append(nodes, *node)
	}

	for i := 0; i < len(data)/2; i++ {
		var level []MerkleNode

		for j := 0; j < len(nodes); j += 2 {
			node := NewMerkleNode(&nodes[j], &nodes[j+1], nil)
			level = append(level, *node)
		}

		nodes = level
	}

	tree := &MerkleTree{RootNode: &nodes[0]}
	return tree
}
