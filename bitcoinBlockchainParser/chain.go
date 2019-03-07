package bitcoinBlockchainParser

import "bytes"

type Chain struct {
	Index  int
	First  *BlockInfo
	Last   *BlockInfo
	Length int
}

func (c *Chain) walkBack( stopAtHash [32]byte ) {
	block := c.Last
	for !bytes.Equal( block.Hash[0:32], stopAtHash[0:32]) {
		oldBlock := block
		block = block.PrevBlockInfo
		block.NextBlockInfo = oldBlock
	}
	c.First = block
}