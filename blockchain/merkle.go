package blockchain

import "crypto/sha256"

type MerkleTree struct {
	RootNode *MerkleNode
}

type MerkleNode struct {
	Left  *MerkleNode
	Right *MerkleNode
	Data  []byte
}

func NewMerkleNode(left, right *MerkleNode, data []byte) *MerkleNode {
	var node MerkleNode

	if left == nil && right == nil {
		hash := sha256.Sum256(data)
		node.Data = hash[:]
	} else {
		prevHashes := append(left.Data, right.Data...)
		hash := sha256.Sum256(prevHashes)
		node.Data = hash[:]
	}

	node.Left = left
	node.Right = right
	return &node
}

func NewMerkleTree(dataArray [][]byte) *MerkleTree {
	var nodes []MerkleNode

	if len(dataArray)%2 != 0 {
		dataArray = append(dataArray, dataArray[len(dataArray)-1])
	}

	for _, data := range dataArray {
		node := NewMerkleNode(nil, nil, data)
		nodes = append(nodes, *node)
	}

	for i := 0; i < len(dataArray)/2; i++ {
		var level []MerkleNode

		for j := 0; j < len(nodes); j += 2 {
			node := NewMerkleNode(&nodes[j], &nodes[j+1], nil)
			level = append(level, *node)
		}
		nodes = level
	}

	tree := MerkleTree{RootNode: &nodes[0]}
	return &tree
}
