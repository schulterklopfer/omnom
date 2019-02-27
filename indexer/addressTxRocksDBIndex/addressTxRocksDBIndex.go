package addressTxRocksDBIndex

import (
	"fmt"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/txscript"
	_ "github.com/mattn/go-sqlite3"
	"github.com/tecbot/gorocksdb"
	"omnom/bitcoinBlockchainParser"
	"time"
)

type AddressTxRocksDBIndex struct {
	//db and statements

	db *gorocksdb.DB

	options *gorocksdb.Options
	readOptions *gorocksdb.ReadOptions
	writeOptions *gorocksdb.WriteOptions
	cfNames []string
	cfHandles []*gorocksdb.ColumnFamilyHandle
	cfOptions []*gorocksdb.Options
	chainCfg *chaincfg.Params
}

func NewAddressTxRocksDBIndex( chainCfg *chaincfg.Params) *AddressTxRocksDBIndex {
	indexer := new(AddressTxRocksDBIndex)
	indexer.options =  gorocksdb.NewDefaultOptions()
	indexer.options.SetCreateIfMissing(true)
	indexer.options.SetErrorIfExists(true)
	indexer.options.SetCreateIfMissingColumnFamilies(true)
	indexer.readOptions = gorocksdb.NewDefaultReadOptions()
	indexer.writeOptions = gorocksdb.NewDefaultWriteOptions()
	indexer.chainCfg = chainCfg
	indexer.cfNames = []string{"default","address","transaction"}
	indexer.cfOptions = []*gorocksdb.Options{indexer.options, indexer.options, indexer.options}
	return indexer
}

func ( indexer *AddressTxRocksDBIndex) OnStart() error {

	var err error

	dbName := fmt.Sprintf("txAddress-%d", time.Now().Unix() )

	db, cfHandles, err := gorocksdb.OpenDbColumnFamilies( indexer.options, dbName, indexer.cfNames, indexer.cfOptions )

	indexer.db = db
	indexer.cfHandles = cfHandles

	if err != nil {
		return err
	}


	return nil
}

func ( indexer *AddressTxRocksDBIndex) OnEnd() error {

	for i:=0; i<len(indexer.cfHandles); i++ {
		indexer.cfHandles[i].Destroy()
	}

	indexer.db.Close()
	return nil

}

func pack( byteArrayArray [][]byte ) []byte {
	result := make( []byte, 0)
	for i:=0; i< len(byteArrayArray); i++ {
		size := byte(len(byteArrayArray[i]))
		result = append(result,size)
		result = append(result,byteArrayArray[i]...)
	}
	return result
}



func ( indexer *AddressTxRocksDBIndex) OnBlock( height int, total int, currentBlock *bitcoinBlockchainParser.Block ) error {
	// anaylse current block

	// insert block into db

	txCount := len(currentBlock.Transactions)
	for i:=0; i<txCount; i++ {

		txOutCount:=len(currentBlock.Transactions[i].Outputs)
		addressBytesArray := make( [][]byte, 0 )

		for j:=0; j<txOutCount; j++ {

			if currentBlock.Transactions[i].Outputs[j].Script == nil ||
				currentBlock.Transactions[i].Outputs[j].Script.Data == nil ||
				len(currentBlock.Transactions[i].Outputs[j].Script.Data) == 0 {
				continue
			}

			_,targetAddresses,_,_ := txscript.ExtractPkScriptAddrs(currentBlock.Transactions[i].Outputs[j].Script.Data, indexer.chainCfg )

			if targetAddresses != nil && len(targetAddresses) > 0 {
				for k:=0; k< len(targetAddresses); k++ {
					addressBytes := []byte(targetAddresses[k].EncodeAddress())
					addressBytesArray = append( addressBytesArray, addressBytes )
					/*
					txs, err := indexer.db.GetCF( indexer.readOptions, indexer.cfHandles[1], addressBytes )
					if err != nil {
						return err
					}
					var newData []byte
					if txs.Size() > 0 {
						newData = append( txs.Data(), currentBlock.Transactions[i].TxId[:]... )
					} else {
						newData = currentBlock.Transactions[i].TxId[:]
					}
					err = indexer.db.PutCF( indexer.writeOptions, indexer.cfHandles[1], addressBytes, newData )
					if err != nil {
						return err
					}
					*/
				}

			}

		}
		if len(addressBytesArray) > 0 {
			txKey := currentBlock.Transactions[i].TxId[:]
			err := indexer.db.PutCF( indexer.writeOptions, indexer.cfHandles[2], txKey, pack(addressBytesArray) )
			if err != nil {
				return err
			}
		}

	}

	/*
	flushOptions := gorocksdb.NewDefaultFlushOptions()
	err := indexer.db.Flush( flushOptions )

	if err != nil {
		return err
	}
	*/

	return nil
}
