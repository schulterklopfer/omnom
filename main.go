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
	"fmt"
	"github.com/btcsuite/btcd/chaincfg"
	"omnom/bitcoinBlockchainParser"
	"omnom/indexer"
	"omnom/indexer/addressTxRocksDBIndex"
	"path"
)

func main() {

	var idx indexer.Indexer
	idx = addressTxRocksDBIndex.NewAddressTxRocksDBIndex(&chaincfg.TestNet3Params)
	//idx = addressTxSqlite3Index.NewAddressTxSqlite3Index(&chaincfg.TestNet3Params)

	//idx = fullSqlite3Index.NewFullSqlite3Index(&chaincfg.TestNet3Params)
	err := idx.OnStart()

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


	/*
	options :=  gorocksdb.NewDefaultOptions()
	options.SetCreateIfMissing(false)
	options.SetErrorIfExists(false)
	options.SetCreateIfMissingColumnFamilies(false)

	readOptions := gorocksdb.NewDefaultReadOptions()
	cfNames := []string{"default","address","transaction"}
	cfOptions := []*gorocksdb.Options{options, options, options}


	db, cfHandles, err := gorocksdb.OpenDbColumnFamilies( options, idx.DBName(), cfNames, cfOptions )

	//txs, err := db.GetCF( readOptions, cfHandles[1], []byte("n3GNqMveyvaPvUbH469vDRadqpJMPc84JA") )

	iter := db.NewIteratorCF(readOptions, cfHandles[1] )

	for iter.SeekToFirst(); iter.Valid(); iter.Next() {
		k := iter.Key()
		v := iter.Value()
		fmt.Printf("%s: %x\n",string(k.Data()),v.Data())

	}
	*/


}