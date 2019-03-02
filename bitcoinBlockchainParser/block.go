package bitcoinBlockchainParser

import (
	"encoding/binary"
	"fmt"
)

type BlockInfo struct {
	Hash [32]byte
	Size uint32
	// Header
	PrevHash [32]byte
	PrevBlock *BlockInfo
	NextBlock *BlockInfo

	BlkFilePosition int32
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
/*
func (b *BlockInfo) ToBytes() ([]byte, error) {

	toSerialize := new( blockInfoSerialized )

	if b.PrevBlock != nil {
		toSerialize.PrevHash = b.PrevBlock.Hash
	}

	if b.NextBlock != nil {
		toSerialize.NextHash = b.NextBlock.Hash
	}

	toSerialize.BlkFileNumber = b.BlkFileNumber
	toSerialize.BlkFilePosition = b.BlkFilePosition

	var buffer bytes.Buffer

	enc := gob.NewEncoder(&buffer)
	err := enc.Encode(toSerialize)

	if err != nil {
		return nil,err
	}

	return buffer.Bytes(), nil
}
*/

func (b *BlockInfo) ToBytes() []byte {
	// dont use varint for sake of simplicity. tradeof: a few more megabytes for an
	// index which has gigabytes in total
	// order: prevHash (32), nextHash (32), fileNumber (16), filePosition (32)
	infoBytes := make( []byte, 32+32+16+32 )
	buffer := make( []byte, 32 )

	if b.PrevBlock != nil {
		copy( infoBytes[0:32], b.PrevBlock.Hash[0:32] )
	}

	if b.NextBlock != nil {
		copy( infoBytes[32:64], b.NextBlock.Hash[0:32] )
	}

	buffer = buffer[0:16]
	binary.LittleEndian.PutUint16( buffer, b.BlkFileNumber )
	copy( infoBytes[64:80], buffer )

	buffer = buffer[0:32]
	binary.LittleEndian.PutUint32( buffer, uint32(b.BlkFilePosition) )
	copy( infoBytes[80:112], buffer )
	return infoBytes
}

func BlockInfoFromBytes( blockHash [32]byte, bytes []byte, blockInfoLookup map[[32]byte]*BlockInfo ) *BlockInfo  {
	blockInfo := new( BlockInfo )

	blockInfo.Hash = blockHash

	var prevBlockHash [32]byte
	copy( prevBlockHash[0:32], bytes[0:32] )

	var nextBlockHash [32]byte
	copy( nextBlockHash[0:32], bytes[32:64] )

	buffer := make( []byte, 64 )

	buffer = buffer[0:16]
	copy( buffer, bytes[64:80] )
	blkFileNumber := binary.LittleEndian.Uint16(buffer)

	buffer = buffer[0:64]
	copy( buffer, bytes[80:112] )
	blkFilePosition := binary.LittleEndian.Uint32(buffer)

	blockInfo.BlkFileNumber = blkFileNumber
	blockInfo.BlkFilePosition = int32(blkFilePosition)

	if bi, ok := blockInfoLookup[prevBlockHash]; ok {
		blockInfo.PrevBlock = bi
	}

	if bi, ok := blockInfoLookup[nextBlockHash]; ok {
		blockInfo.NextBlock = bi
	}

	return blockInfo
}


func (b *BlockInfo) IsGenesis() bool {
	return !b.hasPrev()
}

func (b *BlockInfo) IsTip() bool {
	return b.hasPrev() && !b.hasNext()
}

func (b *BlockInfo) hasPrev() bool {
	return !allZero(b.PrevHash)
}

func (b *BlockInfo) hasNext() bool {
	return b.NextBlock != nil
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

