package main

/*
if you are running on a raspberry pi you will also need to add the "--frymypi" option

so the most unsafe version would be: `cnIndex --reckless --frymypi` (edited)
it will also burn down your house
 */

//https://godoc.org/github.com/btcsuite/btcd/rpcclient
/*
zmqpubrawblock=tcp://0.0.0.0:18501
zmqpubrawtx=tcp://0.0.0.0:18502
 */

import (
	"omnom/bitcoinBlockchainParser"
	"omnom/indexer"
	"omnom/indexer/addressTxSqlite3Index"
	"github.com/btcsuite/btcd/chaincfg"
	"path"
)


func main() {

	var idx indexer.Indexer
	idx = addressTxSqlite3Index.NewAddressTxSqlite3Index(&chaincfg.TestNet3Params)

	err := idx.OnStart()

	if err != nil {
		return
	}

	bp := bitcoinBlockchainParser.NewBitcoinBlockchainParser(path.Join("testnet3", "blocks2"), idx.OnBlock )
	bp.ParseBlocks()
	bp.Close()

	idx.OnEnd()

}