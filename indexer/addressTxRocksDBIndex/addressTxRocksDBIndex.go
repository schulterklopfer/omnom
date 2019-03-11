/*
 * MIT License
 *
 * Copyright (c) 2019 schulterklopfer/SKP
 *
 * Permission is hereby granted, free of charge, to any person obtaining a copy
 * of this software and associated documentation files (the "Software"), to deal
 * in the Software without restriction, including without limitation the rights
 * to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
 * copies of the Software, and to permit persons to whom the Software is
 * furnished to do so, subject to the following conditions:
 *
 * The above copyright notice and this permission notice shall be included in all
 * copies or substantial portions of the Software.
 *
 * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 * IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 * FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 * AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 * LIABILIT * Y, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 * OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
 * SOFTWARE.
 */

package addressTxRocksDBIndex

import (
  "bytes"
  "encoding/binary"
  "fmt"
  "github.com/btcsuite/btcd/chaincfg"
  "github.com/btcsuite/btcd/txscript"
  "github.com/pkg/errors"
  "github.com/tecbot/gorocksdb"
  "log"
  "omnom/bitcoinBlockchainParser"
  "omnom/indexer"
)

type AddressTxRocksDBIndex struct {
  //db and statements
  db               *gorocksdb.DB
  options          *gorocksdb.Options
  readOptions      *gorocksdb.ReadOptions
  writeOptions     *gorocksdb.WriteOptions
  cfNames          []string
  cfHandles        []*gorocksdb.ColumnFamilyHandle
  cfOptions        []*gorocksdb.Options
  chainCfg         *chaincfg.Params
  blockInfoIndex   bool
  addressIndex     bool
  reorgCacheSize   int
  dbName           string
  genesisBlockHash [32]byte
  tipBlockHash     [32]byte
  blockCount       uint64
  indexSearch      *AddressTxRocksDBIndexSearch
}

func NewAddressTxRocksDBIndex(chainCfg *chaincfg.Params) *AddressTxRocksDBIndex {
  indexer := new(AddressTxRocksDBIndex)

  indexer.blockInfoIndex = true
  indexer.addressIndex = true

  indexer.reorgCacheSize = 10 // blocks

  indexer.options = gorocksdb.NewDefaultOptions()
  indexer.options.EnableStatistics()
  indexer.options.SetCreateIfMissing(true)
  indexer.options.SetErrorIfExists(false)
  indexer.options.SetCreateIfMissingColumnFamilies(true)
  indexer.readOptions = gorocksdb.NewDefaultReadOptions()
  indexer.writeOptions = gorocksdb.NewDefaultWriteOptions()
  indexer.writeOptions.DisableWAL(true)
  indexer.writeOptions.SetSync(false)
  indexer.chainCfg = chainCfg
  indexer.cfNames = []string{"default", "address", "transaction", "block", "blockinfo"}
  indexer.cfOptions = []*gorocksdb.Options{indexer.options, indexer.options, indexer.options, indexer.options, indexer.options}
  return indexer
}

func (indexer *AddressTxRocksDBIndex) OnStart() (bool, error) {

  var err error
  existing := true
  indexer.dbName = fmt.Sprintf("address2tx")

  db, cfHandles, err := gorocksdb.OpenDbColumnFamilies(indexer.options, indexer.dbName, indexer.cfNames, indexer.cfOptions)
  if err != nil {
    return false, err
  }

  indexer.db = db
  indexer.cfHandles = cfHandles
  indexer.indexSearch = NewIndexSearch(indexer.db, indexer.readOptions, indexer.cfHandles)

  //check if we have properties stored which tell us
  //that some index is already built
  txs, err := indexer.db.GetCF(indexer.readOptions, indexer.cfHandles[0], []byte("genesisBlockHash"))
  if err != nil {
    return false, err
  }

  if txs.Size() == 32 {
    copy(indexer.genesisBlockHash[0:32], txs.Data())
  } else {
    existing = false
  }
  txs.Free()

  txs, err = indexer.db.GetCF(indexer.readOptions, indexer.cfHandles[0], []byte("tipBlockHash"))
  if err != nil {
    return false, err
  }

  if txs.Size() == 32 {
    copy(indexer.tipBlockHash[0:32], txs.Data())
  } else {
    existing = false
  }
  txs.Free()

  txs, err = indexer.db.GetCF(indexer.readOptions, indexer.cfHandles[0], []byte("blockCount"))
  if err != nil {
    return false, err
  }

  if txs.Size() == 8 {
    indexer.blockCount = binary.LittleEndian.Uint64(txs.Data())
  } else {
    existing = false
  }
  txs.Free()

  return existing, nil
}

func (indexer *AddressTxRocksDBIndex) DBName() string {
  return indexer.dbName
}

func (indexer *AddressTxRocksDBIndex) DB() interface{} {
  return indexer.db
}

func (indexer *AddressTxRocksDBIndex) OnEnd() error {

  for i := 0; i < len(indexer.cfHandles); i++ {
    indexer.cfHandles[i].Destroy()
  }

  indexer.db.Close()

  indexer.options.Destroy()
  indexer.readOptions.Destroy()
  indexer.writeOptions.Destroy()

  return nil

}

func pack(byteArrayArray [][]byte) []byte {
  result := make([]byte, 0)
  for i := 0; i < len(byteArrayArray); i++ {
    size := byte(len(byteArrayArray[i]))
    result = append(result, size)
    result = append(result, byteArrayArray[i]...)
  }
  return result
}

func unpack(bytes []byte) [][]byte {
  result := make([][]byte, 0)
  for i := 0; i < len(bytes); {
    l := int(bytes[i])
    i++
    a := make([]byte, l)
    for j := 0; j < l; j++ {
      a[j] = bytes[i+j]
    }
    i += l
    result = append(result, a)
  }
  return result
}

func (indexer *AddressTxRocksDBIndex) OnBlockInfo(height int, total int, blockInfo *bitcoinBlockchainParser.BlockInfo) error {
  if indexer.blockInfoIndex {
    // store away blockinfo, when historical index is done and new blocks are comming.
    // With this info we will be able to recsontruct the chain to the genesis block
    // and detect reorgs
    blockInfoBytes := blockInfo.ToBytes()
    // write into blockinfo column family: 3
    err := indexer.db.PutCF(indexer.writeOptions, indexer.cfHandles[4], blockInfo.Hash[0:32], blockInfoBytes)
    if err != nil {
      return err
    }

    if blockInfo.IsGenesis() {
      err := indexer.db.PutCF(indexer.writeOptions, indexer.cfHandles[0], []byte("genesisBlockHash"), blockInfo.Hash[0:32])
      if err != nil {
        return err
      }
    } else if blockInfo.IsTip() {
      err := indexer.db.PutCF(indexer.writeOptions, indexer.cfHandles[0], []byte("tipBlockHash"), blockInfo.Hash[0:32])
      if err != nil {
        return err
      }
    }

    if height == total-1 {
      bytes := make([]byte, 8)
      binary.LittleEndian.PutUint64(bytes, uint64(total))
      err := indexer.db.PutCF(indexer.writeOptions, indexer.cfHandles[0], []byte("blockCount"), bytes)
      if err != nil {
        return err
      }
    }

  }
  return nil
}

func (indexer *AddressTxRocksDBIndex) OnBlock(height int, total int, currentBlock *bitcoinBlockchainParser.Block) error {

  if indexer.addressIndex {
    // insert block into db
    txCount := len(currentBlock.Transactions)
    blockTransactionMap := make(map[[32]byte]bool, 0)

    for i := 0; i < txCount; i++ {
      if height > total-indexer.reorgCacheSize {
        blockTransactionMap[currentBlock.Transactions[i].TxId] = true
      }
      txOutCount := len(currentBlock.Transactions[i].Outputs)
      transactionAddressMap := make(map[string]bool, 0)

      // todo: check inputs for "source addies" and add 1 or many flags bytes to mark if address/tx is from input or output
      // also add varint for number of addies/tx
      if indexer.addressIndex {
        for j := 0; j < txOutCount; j++ {

          if currentBlock.Transactions[i].Outputs[j].Script == nil ||
              currentBlock.Transactions[i].Outputs[j].Script.Data == nil ||
              len(currentBlock.Transactions[i].Outputs[j].Script.Data) == 0 {
            continue
          }

          _, targetAddresses, _, _ := txscript.ExtractPkScriptAddrs(currentBlock.Transactions[i].Outputs[j].Script.Data, indexer.chainCfg)

          if targetAddresses != nil && len(targetAddresses) > 0 {
            for k := 0; k < len(targetAddresses); k++ {
              address := targetAddresses[k].EncodeAddress()
              if height >= total-indexer.reorgCacheSize {
                transactionAddressMap[address] = true
              }

              if indexer.addressIndex {
                txs, err := indexer.db.GetCF(indexer.readOptions, indexer.cfHandles[1], []byte( address ))
                if err != nil {
                  return err
                }

                var newData []byte

                if txs.Size() > 0 {
                  newData := make([]byte, txs.Size()+32)
                  copy(newData[0:txs.Size()], txs.Data())
                  copy(newData[txs.Size():txs.Size()+32], currentBlock.Transactions[i].TxId[0:32])
                } else {
                  newData = currentBlock.Transactions[i].TxId[0:32]
                }

                txs.Free()

                err = indexer.db.PutCF(indexer.writeOptions, indexer.cfHandles[1], []byte( address ), newData)
                if err != nil {
                  return err
                }
              }
            }
          }
        }
      }
      if height >= total-indexer.reorgCacheSize {
        addressBytesArray := make([][]byte, 0)
        for k := range transactionAddressMap {
          addressBytesArray = append(addressBytesArray, []byte(k))
        }

        err := indexer.db.PutCF(indexer.writeOptions, indexer.cfHandles[2], currentBlock.Transactions[i].TxId[:], pack(addressBytesArray))
        if err != nil {
          return err
        }
      }

      if height >= total-indexer.reorgCacheSize {
        txCount := len(blockTransactionMap)
        transactionsBytes := make([]byte, txCount*32)
        index := 0
        for k := range blockTransactionMap {
          copy(transactionsBytes[index*32:index*32+32], k[0:32])
          index++
        }
        err := indexer.db.PutCF(indexer.writeOptions, indexer.cfHandles[3], currentBlock.Hash[0:32], transactionsBytes)
        if err != nil {
          return err
        }
      }

    }
  }
  return nil
}

func (indexer *AddressTxRocksDBIndex) ShouldParseBlockInfo() bool {
  return indexer.blockInfoIndex
}

func (indexer *AddressTxRocksDBIndex) ShouldParseBlockBody() bool {
  return indexer.addressIndex
}

func (indexer *AddressTxRocksDBIndex) GetGenesisBlockInfo() (*bitcoinBlockchainParser.BlockInfo, error) {
  return indexer.indexSearch.FindBlockInfoByBlockHash(indexer.genesisBlockHash[0:32])
}

func (indexer *AddressTxRocksDBIndex) GetTipBlockInfo() (*bitcoinBlockchainParser.BlockInfo, error) {
  return indexer.indexSearch.FindBlockInfoByBlockHash(indexer.tipBlockHash[0:32])
}

func (indexer *AddressTxRocksDBIndex) GetBlockCount() uint64 {
  return indexer.blockCount
}

func (indexer *AddressTxRocksDBIndex) IndexSearch() indexer.IndexSearch {
  return indexer.indexSearch
}

func (indexer *AddressTxRocksDBIndex) CheckBlockInfoEntries(longestChain *bitcoinBlockchainParser.Chain) error {
  if !bytes.Equal(longestChain.Last.Hash[0:32], indexer.genesisBlockHash[0:32]) {
    return errors.New("Last block mismatch")
  }
  if bytes.Equal(longestChain.First.Hash[0:32], indexer.tipBlockHash[0:32]) {
    // genesis block and tip are the same:
    // walk through chain and check if it matches the data
    // in the index
    block := longestChain.First
    log.Println("Last block and tip are the same. Comparing data")
    for !block.IsGenesis() {
      bi, err := indexer.indexSearch.FindBlockInfoByBlockHash(block.Hash[0:32])

      if err != nil {
        return nil
      }

      if !bytes.Equal(block.Hash[0:32], bi.Hash[0:32]) ||
          !bytes.Equal(block.PrevHash[0:32], bi.PrevHash[0:32]) {
        return errors.New("Chain in index doesn't match chain on disk")
      }

      block = block.PrevBlockInfo
    }
    log.Println("Looks good to me.")

  } else {
    // analyse who is ahead
  }
  return nil
}

func (indexer *AddressTxRocksDBIndex) CleanupReorgCache(longestChain *bitcoinBlockchainParser.Chain) error {
  if !bytes.Equal(longestChain.First.Hash[0:32], indexer.tipBlockHash[0:32]) {
    return errors.New("Chain tip mismatch")
  }
  blockInfo := longestChain.Last
  log.Println("Cleaning up reorg cache")
  counter := 0
  for blockInfo.PrevBlockInfo != nil {
    if counter > indexer.reorgCacheSize {
      // remove stuff here
      txids, err := indexer.indexSearch.FindTransactionIdsByBlockHash(blockInfo.Hash[0:32])
      if err == nil && txids == nil {
        // assume everything b4 was also deleted and break
        break
      }

      if txids != nil {
        for t := 0; t < len(txids); t++ {
          err = indexer.db.DeleteCF(indexer.writeOptions, indexer.cfHandles[2], txids[t][0:32])
          if err != nil {
            log.Printf("Error when deleting txid %x: %s", txids[t][0:32], err)
          }
        }
      }

      err = indexer.db.DeleteCF(indexer.writeOptions, indexer.cfHandles[3], blockInfo.Hash[0:32])
      if err != nil {
        log.Printf("Error when deleting blockInfo %x: %s", blockInfo.Hash[0:32], err)
      }

    }
    blockInfo = blockInfo.PrevBlockInfo
    counter++
  }

  return nil

}

func (indexer *AddressTxRocksDBIndex) GetReorgCacheSize() int {
  return indexer.reorgCacheSize
}
