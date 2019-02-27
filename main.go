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
 */import (
	"encoding/hex"
	"fmt"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/txscript"
	"omnom/bitcoinBlockchainParser"
	"omnom/indexer"
	"omnom/indexer/addressTxRocksDBIndex"
	"path"
)

func main() {
	//1456655
	//tb1qdzytarlc83kgqxn32rcr0sj4rrp47ektctma0p
	//local:
	//spkString := "76a9146942d8a8539c7898ae73062394e70193e04048c888ac"
	//blockstream.info:
	spkString := "76a914f90782888b23f34ba4ad66baa6cd3d240df279f188ac"
	spk, err := hex.DecodeString( spkString )

	cls, addresses, foo, err := txscript.ExtractPkScriptAddrs(spk,&chaincfg.TestNet3Params)


	for i:=0; i< len(addresses); i++ {
		fmt.Println( addresses[i] )
	}

	fmt.Println( spk, err, cls, addresses, foo )



	var idx indexer.Indexer
	idx = addressTxRocksDBIndex.NewAddressTxRocksDBIndex(&chaincfg.TestNet3Params)
	//idx = addressTxSqlite3Index.NewAddressTxSqlite3Index(&chaincfg.TestNet3Params)

	//idx = fullSqlite3Index.NewFullSqlite3Index(&chaincfg.TestNet3Params)
	err = idx.OnStart()

	if err != nil {
		fmt.Println(err)
		return
	}

	bp := bitcoinBlockchainParser.NewBitcoinBlockchainParser(path.Join("testnet3", "blocks"), idx.OnBlock )
	err = bp.ParseBlocks()
	if err != nil {
		fmt.Println(err)
	}
	bp.Close()

	idx.OnEnd()


}