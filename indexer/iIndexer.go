package indexer

import "omnom/bitcoinBlockchainParser"

type Indexer interface {
	OnStart() (bool,error)
	OnEnd() error
	OnBlockInfo( height int, blockCount int, blockInfo *bitcoinBlockchainParser.BlockInfo ) error
	OnBlock( height int, blockCount int, block *bitcoinBlockchainParser.Block ) error
	DBName() string

	ShouldParseBlockInfo() bool
	ShouldParseBlockBody() bool

	CheckBlockInfoEntries( *bitcoinBlockchainParser.Chain ) error
}