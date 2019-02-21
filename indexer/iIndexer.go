package indexer

import "cyphernode_indexer/bitcoinBlockchainParser"

type Indexer interface {
	OnStart() error
	OnEnd() error
	OnBlock( height int, blockCount int, block *bitcoinBlockchainParser.Block )
}