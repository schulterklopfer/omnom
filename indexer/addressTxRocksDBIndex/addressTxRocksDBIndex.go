package addressTxRocksDBIndex

import (
	"encoding/binary"
	"fmt"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/txscript"
	_ "github.com/mattn/go-sqlite3"
	"github.com/tecbot/gorocksdb"
	"omnom/bitcoinBlockchainParser"
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

	blockInfoIndex bool
	transactionIndex bool
	addressIndex bool
	blockIndex bool

	dbName string

	genesisBlockHash [32]byte
	tipBlockHash [32]byte
	blockCount uint64
}

func NewAddressTxRocksDBIndex( chainCfg *chaincfg.Params) *AddressTxRocksDBIndex {
	indexer := new(AddressTxRocksDBIndex)

	indexer.blockInfoIndex = true
	indexer.transactionIndex = true
	indexer.addressIndex = true
	indexer.blockIndex = true

	indexer.options = gorocksdb.NewDefaultOptions()
	indexer.options.SetCreateIfMissing(true)
	indexer.options.SetErrorIfExists(false)
	indexer.options.SetCreateIfMissingColumnFamilies(true)
	indexer.readOptions = gorocksdb.NewDefaultReadOptions()
	indexer.writeOptions = gorocksdb.NewDefaultWriteOptions()
	indexer.writeOptions.DisableWAL(true)
	indexer.writeOptions.SetSync(false)
	indexer.chainCfg = chainCfg
	indexer.cfNames = []string{"default","address","transaction","block","blockinfo"}
	indexer.cfOptions = []*gorocksdb.Options{indexer.options, indexer.options, indexer.options, indexer.options, indexer.options}
	return indexer
}

func ( indexer *AddressTxRocksDBIndex) OnStart() (bool,error) {

	var err error
	existing := true
	indexer.dbName = fmt.Sprintf("address2tx" )

	db, cfHandles, err := gorocksdb.OpenDbColumnFamilies( indexer.options, indexer.dbName, indexer.cfNames, indexer.cfOptions )
	if err != nil {
		return false,err
	}

	indexer.db = db
	indexer.cfHandles = cfHandles

	//check if we have properties stored which tell us
	//that some index is already built
	txs, err := indexer.db.GetCF( indexer.readOptions, indexer.cfHandles[0], []byte("genesisBlockHash") )
	if err != nil {
		return false,err
	}

	if txs.Size() == 32  {
		copy(indexer.genesisBlockHash[0:32], txs.Data())
	} else {
		existing = false
	}
	txs.Free()


	txs, err = indexer.db.GetCF( indexer.readOptions, indexer.cfHandles[0], []byte("tipBlockHash") )
	if err != nil {
		return false,err
	}

	if txs.Size() == 32  {
		copy(indexer.tipBlockHash[0:32], txs.Data())
	} else {
		existing = false
	}
	txs.Free()

	txs, err = indexer.db.GetCF( indexer.readOptions, indexer.cfHandles[0], []byte("blockCount") )
	if err != nil {
		return false,err
	}

	if txs.Size() == 8  {
		binary.LittleEndian.Uint64(txs.Data())
	} else {
		existing = false
	}
	txs.Free()

	return existing,nil
}

func ( indexer *AddressTxRocksDBIndex) DBName() string {
	return indexer.dbName
}

func ( indexer *AddressTxRocksDBIndex) OnEnd() error {

	for i:=0; i<len(indexer.cfHandles); i++ {
		indexer.cfHandles[i].Destroy()
	}

	indexer.db.Close()

	indexer.options.Destroy()
	indexer.readOptions.Destroy()
	indexer.writeOptions.Destroy()

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

func unpack( bytes []byte ) [][]byte {
	result := make( [][]byte, 0)
	for i:=0; i < len(bytes); {
		l := int(bytes[i])
		i++
		a := make( []byte, l )
		for j:=0; j<l; j++ {
			a[j]=bytes[i+j]
		}
		i+=l
		result = append( result, a )
	}
	return result
}

func ( indexer *AddressTxRocksDBIndex) OnBlockInfo( height int, total int, blockInfo *bitcoinBlockchainParser.BlockInfo ) error {
	if indexer.blockInfoIndex {
		// store away blockinfo, when historical index is done and new blocks are comming.
		// With this info we will be able to recsontruct the chain to the genesis block
		// and detect reorgs
		blockInfoBytes := blockInfo.ToBytes()
		// write into blockinfo column family: 3
		err := indexer.db.PutCF( indexer.writeOptions, indexer.cfHandles[4], blockInfo.Hash[0:32], blockInfoBytes )
		if err != nil {
			return err
		}

		if blockInfo.IsGenesis() {
			err := indexer.db.PutCF( indexer.writeOptions, indexer.cfHandles[0], []byte("genesisBlockHash"), blockInfo.Hash[0:32] )
			if err != nil {
				return err
			}
		} else if blockInfo.IsTip() {
			err := indexer.db.PutCF( indexer.writeOptions, indexer.cfHandles[0], []byte("tipBlockHash"), blockInfo.Hash[0:32] )
			if err != nil {
				return err
			}
		}

		if height == total {
			bytes := make( []byte,8)
			binary.LittleEndian.PutUint64(bytes,uint64(total))
			err := indexer.db.PutCF( indexer.writeOptions, indexer.cfHandles[0], []byte("blockCount"), bytes )
			if err != nil {
				return err
			}
		}


	}
	return nil
}

func ( indexer *AddressTxRocksDBIndex) OnBlock( height int, total int, currentBlock *bitcoinBlockchainParser.Block ) error {

	if indexer.addressIndex || indexer.transactionIndex || indexer.blockIndex {
		// insert block into db
		txCount := len(currentBlock.Transactions)
		blockTransactionMap := make( map[[32]byte]bool, 0 )

		for i:=0; i<txCount; i++ {
			if indexer.blockIndex {
				blockTransactionMap[currentBlock.Transactions[i].TxId]=true
			}
			txOutCount:=len(currentBlock.Transactions[i].Outputs)
			transactionAddressMap := make( map[string]bool, 0 )

			// todo: check inputs for "source addies" and add 1 or many flags bytes to mark if address/tx is from input or output
			// also add varint for number of addies/tx
			if indexer.addressIndex || indexer.transactionIndex {
				for j:=0; j<txOutCount; j++ {

					if currentBlock.Transactions[i].Outputs[j].Script == nil ||
						currentBlock.Transactions[i].Outputs[j].Script.Data == nil ||
						len(currentBlock.Transactions[i].Outputs[j].Script.Data) == 0 {
						continue
					}

					_,targetAddresses,_,_ := txscript.ExtractPkScriptAddrs(currentBlock.Transactions[i].Outputs[j].Script.Data, indexer.chainCfg )

					if targetAddresses != nil && len(targetAddresses) > 0 {
						for k:=0; k< len(targetAddresses); k++ {
							address := targetAddresses[k].EncodeAddress()
							if indexer.transactionIndex {
								transactionAddressMap[address]=true
							}

							if indexer.addressIndex {
								txs, err := indexer.db.GetCF( indexer.readOptions, indexer.cfHandles[1], []byte( address ))
								if err != nil {
									return err
								}

								var newData []byte

								if txs.Size() > 0  {
									newData := make( []byte, txs.Size()+32 )
									copy(newData[0:txs.Size()],txs.Data())
									copy(newData[txs.Size():txs.Size()+32], currentBlock.Transactions[i].TxId[0:32] )
								} else {
									newData = currentBlock.Transactions[i].TxId[0:32]
								}

								txs.Free()

								err = indexer.db.PutCF( indexer.writeOptions, indexer.cfHandles[1], []byte( address ), newData )
								if err != nil {
									return err
								}
							}
						}
					}
				}
			}
			if indexer.transactionIndex {
				addressBytesArray := make([][]byte,0)
				for k := range transactionAddressMap {
					addressBytesArray = append(addressBytesArray, []byte(k))
				}

				err := indexer.db.PutCF( indexer.writeOptions, indexer.cfHandles[2], currentBlock.Transactions[i].TxId[:], pack(addressBytesArray) )
				if err != nil {
					return err
				}
			}

			if indexer.blockIndex {
				txCount := len(blockTransactionMap)
				transactionsBytes := make([]byte,txCount*32)
				index := 0
				for k := range blockTransactionMap {
					copy( transactionsBytes[index*32:index*32+32], k[0:32])
					index++
				}
				err := indexer.db.PutCF( indexer.writeOptions, indexer.cfHandles[3], currentBlock.Hash[0:32], transactionsBytes )
				if err != nil {
					return err
				}
			}

		}
	}


	return nil
}

func ( indexer *AddressTxRocksDBIndex) ShouldParseBlockInfo() bool {
	return indexer.blockInfoIndex
}

func ( indexer *AddressTxRocksDBIndex) ShouldParseBlockBody() bool {
	return indexer.transactionIndex || indexer.addressIndex
}

func ( indexer *AddressTxRocksDBIndex) CheckBlockInfoEntries( longestChain *bitcoinBlockchainParser.Chain ) error {
	return nil
}
