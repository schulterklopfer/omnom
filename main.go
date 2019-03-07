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
  "log"
  "omnom/bitcoinBlockchainParser"
  "omnom/indexer"
  "omnom/indexer/addressTxRocksDBIndex"
  "path"
)

func main() {

  var idx indexer.Indexer
  idx = addressTxRocksDBIndex.NewAddressTxRocksDBIndex(&chaincfg.TestNet3Params)
  existing, err := idx.OnStart()

  //existing = false

  if err != nil {
    fmt.Println(err)
    return
  }

  bp := bitcoinBlockchainParser.NewBitcoinBlockchainParser(path.Join("testnet3_157", "blocks"), idx.OnBlockInfo, idx.OnBlock)

  if !existing {
    // parse historic data
    log.Println("Starting to build index")
    opts := bitcoinBlockchainParser.NewBitcoinBlockchainParserDefaultOptions()
    opts.CallBlockCallback = false
    err = bp.ParseBlocks(opts)

    if err != nil {
      fmt.Println(err)
    }
  } else {
    log.Println("Found index...")
    log.Println("Checking index consistency...")

    tipBlockInfo, err := idx.GetTipBlockInfo()

    // walk back index blockInfo to genesis block
    opts := bitcoinBlockchainParser.NewBitcoinBlockchainParserDefaultOptions()
    opts.BlkFilePosition = tipBlockInfo.BlkFilePosition
    opts.BlkFileNumber = tipBlockInfo.BlkFileNumber
    opts.StopAtHash = tipBlockInfo.Hash
    opts.StartBlockHeight = idx.GetBlockCount() - 1

    // look for chains with current tip as root
    blockMap, blockOrder, err := bp.CollectBlockInfo(opts)
    if err != nil {
      log.Fatalf("Error in collecting block info: %s", err)
      return
    }

    chains, err := bp.FindChains(blockMap, blockOrder, opts)

    if err != nil && len(chains) == 0 {
      // this means there is no data in the blk files
      // leading back to the current tip in the index. -> Reorg
      // we need to walk back the indexed chain and check for
      // blocks connecting to a previous tip. If we find a chain
      // linked to a previous tip, we need to remove all indexed
      // data for the dangling chain and then update the reorg cache

    }

    log.Println(chains)

    log.Printf("Last tip:    %x\n", tipBlockInfo.Hash)
    log.Printf("Chain first: %x\n", chains[0].First.Hash)
    log.Printf("Chain last:  %x\n", chains[0].Last.Hash)
    log.Printf("Total len:   %d\n", int(chains[0].Length)+int(opts.StartBlockHeight))
    //longestChain := chains[0]

    //err = idx.CleanupReorgCache( longestChain )

    //if err != nil {
    //	log.Fatalf( "Error in checking index consistency: %s", err )
    //}
  }

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
