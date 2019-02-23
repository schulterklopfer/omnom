package bitcoinBlockchainParser


type Chain struct {
	Index int
	Tip *BlockInfo
	Genesis *BlockInfo
	Length int
}

func (c *Chain) walkBack() {
	block := c.Tip
	for !block.isGenesis() {
		oldBlock := block
		block = block.PrevBlock
		block.NextBlock = oldBlock
	}
	c.Genesis = block
}