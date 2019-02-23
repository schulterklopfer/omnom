package bitcoinBlockchainParser

import "fmt"

type Block struct {
	Hash [32]byte
	Size uint32
	// Header
	Version uint32
	PrevHash [32]byte
	PrevBlock *Block
	NextBlock *Block
	MerkleRoot [32]byte
	Timestamp uint32
	Difficulty [4]byte
	Nonce uint32

	//Transactions
	Transactions []Transaction

	BlkFilePosition int64
	BlkFileNumber uint

	PartOfChain bool
}

func (b *Block) HashString() string {
	return fmt.Sprintf("%x", b.Hash )
}

func (b *Block) isGenesis() bool {
	return !b.hasPrev()
}

func (b *Block) hasPrev() bool {
	return !allZero(b.PrevHash)
}

func (b *Block) isPrevTo( block *Block ) bool {
	if b == nil || block == nil {
		return false
	}
	for i:=0; i<32; i++ {
		if b.Hash[i] != block.PrevHash[i] {
			return false
		}
	}
	return true
}

func (b *Block) isEqualTo( block *Block ) bool {
	if b == nil || block == nil {
		return false
	}
	for i:=0; i<32; i++ {
		if b.Hash[i] != block.Hash[i] {
			return false
		}
	}
	return true
}

