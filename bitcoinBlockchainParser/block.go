package bitcoinBlockchainParser

import "fmt"

type BlockInfo struct {
	Hash [32]byte
	Size uint32
	// Header
	PrevHash [32]byte
	PrevBlock *BlockInfo
	NextBlock *BlockInfo

	BlkFilePosition int64
	BlkFileNumber uint16

	PartOfChain bool
}

type Block struct {
	Hash [32]byte
	Size uint32
	// Header
	Version uint32
	PrevHash [32]byte
	MerkleRoot [32]byte
	Timestamp uint32
	Difficulty [4]byte
	Nonce uint32

	//Transactions
	Transactions []Transaction
}

func (b *Block) HashString() string {
	return fmt.Sprintf("%x", b.Hash )
}

func (b *BlockInfo) isGenesis() bool {
	return !b.hasPrev()
}

func (b *BlockInfo) hasPrev() bool {
	return !allZero(b.PrevHash)
}

func (b *BlockInfo) isPrevTo( block *Block ) bool {
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

func (b *BlockInfo) isEqualTo( block *Block ) bool {
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

