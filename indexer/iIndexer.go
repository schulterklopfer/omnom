package indexer

import (
	"omnom/bitcoinBlockchainParser"
)

type Indexer interface {
	OnStart() (bool,error)
	OnEnd() error
	OnBlockInfo( height int, blockCount int, blockInfo *bitcoinBlockchainParser.BlockInfo ) error
	OnBlock( height int, blockCount int, block *bitcoinBlockchainParser.Block ) error
	DBName() string

	GetGenesisBlockInfo() (*bitcoinBlockchainParser.BlockInfo, error)
	GetTipBlockInfo() (*bitcoinBlockchainParser.BlockInfo, error)
	GetBlockCount() uint64

	ShouldParseBlockInfo() bool
	ShouldParseBlockBody() bool

	CheckBlockInfoEntries( *bitcoinBlockchainParser.Chain ) error
	CleanupReorgCache( *bitcoinBlockchainParser.Chain ) error

	IndexSearch() IndexSearch
}