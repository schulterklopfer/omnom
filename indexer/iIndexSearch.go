package indexer

import (
	"omnom/bitcoinBlockchainParser"
)

type IndexSearch interface {
	FindTransactionIdsByAddress( address string ) ([][]byte,error)
	FindAddressesByTransactionId( txid string ) ([][]byte,error)
	FindTransactionIdsByBlockHash( blockHash []byte ) ([][32]byte,error)
	FindTransactionIdsByBlockHeight( blockHeight int ) ([][]byte,error)
	FindBlockHashByBlockHeight( blockHeight int ) ([]byte,error)
	FindBlockInfoByBlockHash( blockHash []byte  ) (*bitcoinBlockchainParser.BlockInfo, error)
}
