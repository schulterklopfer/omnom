package indexer

import "omnom/bitcoinBlockchainParser"

type Indexer interface {
	OnStart() error
	OnEnd() error
	OnBlock( height int, blockCount int, block *bitcoinBlockchainParser.Block )
}