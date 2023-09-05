package main

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"log"
)

type Block struct {
	Block    string   `json:"block"`
	Subtrees []string `json:"subtrees"`
}

func NewBlockchain() *Block {
	subtreeHashes := make([]string, 0)
	return &Block{Subtrees: subtreeHashes}
}

func generateRandomHash() string {
	bytes := make([]byte, 32)
	_, err := rand.Read(bytes)
	if err != nil {
		log.Fatal(err)
	}
	return hex.EncodeToString(bytes)
}

func (b *Block) GenerateSubtree() string {
	subtreeHash := generateRandomHash()
	b.Subtrees = append(b.Subtrees, subtreeHash)
	return subtreeHash
}

func (b *Block) GenerateBlock() string {
	blockHash := generateRandomHash()
	b.Block = blockHash
	return blockHash
}

func (b *Block) ToJson() ([]byte, error) {
	j, err := json.Marshal(b)
	if err != nil {
		return nil, err
	}
	return j, nil
}

func (b *Block) Reset() {
	b.Block = ""
	b.Subtrees = make([]string, 0)
}
