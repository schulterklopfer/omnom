/*
 * MIT License
 *
 * Copyright (c) 2019 schulterklopfer/SKP
 *
 * Permission is hereby granted, free of charge, to any person obtaining a copy
 * of this software and associated documentation files (the "Software"), to deal
 * in the Software without restriction, including without limitation the rights
 * to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
 * copies of the Software, and to permit persons to whom the Software is
 * furnished to do so, subject to the following conditions:
 *
 * The above copyright notice and this permission notice shall be included in all
 * copies or substantial portions of the Software.
 *
 * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 * IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 * FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 * AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 * LIABILIT * Y, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 * OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
 * SOFTWARE.
 */

package bitcoinBlockchainParser

import (
  "encoding/binary"
  "fmt"
)

type BlockInfo struct {
  Hash [32]byte
  Size uint32
  // Header
  PrevHash      [32]byte
  PrevBlockInfo *BlockInfo
  NextBlockInfo *BlockInfo

  BlkFilePosition int32
  BlkFileNumber   uint16

  PartOfChain bool
}

type Block struct {
  Hash         [32]byte
  Size         uint32
  Version      uint32
  PrevHash     [32]byte
  MerkleRoot   [32]byte
  Timestamp    uint32
  Difficulty   [4]byte
  Nonce        uint32
  Transactions []Transaction
}

func (b *Block) HashString() string {
  return fmt.Sprintf("%x", b.Hash)
}

/*
func (b *BlockInfo) ToBytes() ([]byte, error) {

	toSerialize := new( blockInfoSerialized )

	if b.PrevBlockInfo != nil {
		toSerialize.PrevHash = b.PrevBlockInfo.Hash
	}

	if b.NextBlockInfo != nil {
		toSerialize.NextHash = b.NextBlockInfo.Hash
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
  infoBytes := make([]byte, 32+32+16+32)
  buffer := make([]byte, 32)

  if b.PrevBlockInfo != nil {
    copy(infoBytes[0:32], b.PrevBlockInfo.Hash[0:32])
  }

  if b.NextBlockInfo != nil {
    copy(infoBytes[32:64], b.NextBlockInfo.Hash[0:32])
  }

  buffer = buffer[0:16]
  binary.LittleEndian.PutUint16(buffer, b.BlkFileNumber)
  copy(infoBytes[64:80], buffer)

  buffer = buffer[0:32]
  binary.LittleEndian.PutUint32(buffer, uint32(b.BlkFilePosition))
  copy(infoBytes[80:112], buffer)
  return infoBytes
}

func BlockInfoFromBytes(blockHash []byte, bytes []byte, blockInfoLookup map[[32]byte]*BlockInfo) *BlockInfo {
  blockInfo := new(BlockInfo)

  copy(blockInfo.Hash[0:32], blockHash)
  copy(blockInfo.PrevHash[0:32], bytes[0:32])

  buffer := make([]byte, 64)

  buffer = buffer[0:16]
  copy(buffer, bytes[64:80])
  blkFileNumber := binary.LittleEndian.Uint16(buffer)

  buffer = buffer[0:64]
  copy(buffer, bytes[80:112])
  blkFilePosition := binary.LittleEndian.Uint32(buffer)

  blockInfo.BlkFileNumber = blkFileNumber
  blockInfo.BlkFilePosition = int32(blkFilePosition)

  if blockInfoLookup != nil {

    if bi, ok := blockInfoLookup[blockInfo.PrevHash]; ok {
      blockInfo.PrevBlockInfo = bi
    }

    var nextBlockHash [32]byte
    copy(nextBlockHash[0:32], bytes[32:64])

    if bi, ok := blockInfoLookup[nextBlockHash]; ok {
      blockInfo.NextBlockInfo = bi
    }
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
  return b.NextBlockInfo != nil
}

func (b *BlockInfo) isPrevTo(block *Block) bool {
  if b == nil || block == nil {
    return false
  }
  for i := 0; i < 32; i++ {
    if b.Hash[i] != block.PrevHash[i] {
      return false
    }
  }
  return true
}

func (b *BlockInfo) isEqualTo(block *Block) bool {
  if b == nil || block == nil {
    return false
  }
  for i := 0; i < 32; i++ {
    if b.Hash[i] != block.Hash[i] {
      return false
    }
  }
  return true
}
