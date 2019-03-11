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

package bitcoinBlockchainParser

import (
  "bytes"
  "crypto/sha256"
  "encoding/binary"
  "fmt"
  "github.com/btcsuite/btcd/chaincfg"
  "github.com/pkg/errors"
  "io/ioutil"
  "math"
  "os"
  "path"
  "sort"
  "strings"
  "time"
)

type BitcoinBlockchainParser struct {
  // private
  directory   string
  onBlockInfo OnBlockInfoCallback
  onBlock     OnBlockCallback
}

type BitcoinBlockchainParserOptions struct {
  // private
  CallBlockInfoCallback bool
  CallBlockCallback     bool
  StopAtPrevHash        [32]byte
  BlkFilePosition       int32
  BlkFileNumber         uint16
  StartBlockHeight      uint64
}

type OnBlockInfoCallback func(int, int, *BlockInfo) error
type OnBlockCallback func(int, int, *Block) error

func NewBitcoinBlockchainParser(directory string, onBlockInfo OnBlockInfoCallback, onBlock OnBlockCallback) *BitcoinBlockchainParser {
  return &BitcoinBlockchainParser{directory, onBlockInfo, onBlock}
}

func NewBitcoinBlockchainParserDefaultOptions() *BitcoinBlockchainParserOptions {
  o := new(BitcoinBlockchainParserOptions)
  o.CallBlockCallback = true
  o.CallBlockInfoCallback = true
  return o
}

func allZero(hash [32]byte) bool {
  for i := 0; i < 32; i++ {
    if hash[i] != 0x00 {
      return false
    }
  }
  return true
}

func filterBlockDataFiles(fileInfos []os.FileInfo) (ret []os.FileInfo) {
  for i := 0; i < len(fileInfos); i++ {
    if strings.HasPrefix(fileInfos[i].Name(), "blk") && strings.HasSuffix(fileInfos[i].Name(), ".dat") {
      ret = append(ret, fileInfos[i])
    }
  }
  return
}

var POSITION_IN_FILE int
var FILE_INDEX int
var CHAINCFG = &chaincfg.TestNet3Params

// todo: use standard length buffers for 4,8,32 and only alloc for variable lengths exceeding 4096 bytes
var buffer1 = make([]byte, 1)
var buffer2 = make([]byte, 2)
var buffer4 = make([]byte, 4)
var buffer8 = make([]byte, 8)
var buffer32 = make([]byte, 32)
var buffer80 = make([]byte, 80)

func (bc *BitcoinBlockchainParser) CollectBlockInfo(options *BitcoinBlockchainParserOptions) (map[[32]byte]*BlockInfo, []*BlockInfo, error) {
  fileInfos, err := ioutil.ReadDir(bc.directory)
  if err != nil {
    return nil, nil, err
  }

  fileInfos = filterBlockDataFiles(fileInfos)
  start := time.Now()
  blockHeight := options.StartBlockHeight

  blockOrder := make([]*BlockInfo, 0)
  blockMap := make(map[[32]byte]*BlockInfo)

  startPositionInFile := options.BlkFilePosition

  for index, fileInfo := range fileInfos {
    var blkFileNumber uint16
    fmt.Sscanf(fileInfo.Name(), "blk%d.dat", &blkFileNumber)

    if blkFileNumber < options.BlkFileNumber {
      continue
    }

    fmt.Printf("Opening %s [%d of %d]\n", fileInfo.Name(), index+1, len(fileInfos))

    // Open readonly
    file, err := os.Open(path.Join(bc.directory, fileInfo.Name()))
    if err != nil {
      return nil, nil, err
    }
    POSITION_IN_FILE = int(startPositionInFile)
    //fmt.Println(err, reader.Size(), fileInfo.Size() )
    nextBlockPosition := int64(POSITION_IN_FILE)
    _, err = file.Seek(nextBlockPosition, 0)
    if err != nil {
      return nil, nil, err
    }
    for nextBlockPosition < fileInfo.Size() {
      blockIndex, err := bc.parseBlockInfo(file)
      if blockIndex == nil {
        break
      }

      blockIndex.BlkFileNumber = blkFileNumber
      blockIndex.BlkFilePosition = int32(nextBlockPosition)

      blockOrder = append(blockOrder, blockIndex)
      blockMap[blockIndex.Hash] = blockIndex

      nextBlockPosition += int64(blockIndex.Size) + 8
      if nextBlockPosition >= fileInfo.Size() {
        nextBlockPosition = fileInfo.Size()
      }
      _, err = file.Seek(nextBlockPosition, 0)

      if err != nil {
        break
      }

      blockHeight++

    }
    startPositionInFile = 0
    elapsed := time.Since(start)
    nanosPerBlock := elapsed.Nanoseconds() / int64(blockHeight-options.StartBlockHeight)
    fmt.Printf("Traversing 1 block took: %s\n", time.Duration(nanosPerBlock))
    fmt.Printf("Traversing %d blocks took: %s\n", blockHeight-options.StartBlockHeight, elapsed)
    fmt.Printf("Done: %d of %d bytes, %d blocks \n", nextBlockPosition, fileInfo.Size(), blockHeight-options.StartBlockHeight)
    fmt.Println("---")

    file.Close()
  }
  elapsed := time.Since(start)
  fmt.Printf("Traversing %d blocks took: %s\n", blockHeight-options.StartBlockHeight, elapsed)
  return blockMap, blockOrder, nil
}

func (bc *BitcoinBlockchainParser) FindChains(blockMap map[[32]byte]*BlockInfo, blockOrder []*BlockInfo, options *BitcoinBlockchainParserOptions) ([]*Chain, error) {
  fmt.Println("\nlooking for chain")

  chains := make([]*Chain, 0)
  if len(blockOrder) == 1 {
    // no new blocks
    chain := new(Chain)
    chain.Index = 0
    chain.Last = blockOrder[0]
    chain.First = blockOrder[0]
    chain.Length = 1
    blockOrder[0].PartOfChain = true
    chains = append(chains, chain)
    return chains,nil
  }

  for i := len(blockOrder) - 1; i >= 0; i-- {
    currentBlock := blockOrder[i]

    // is this block part of another chain?
    // if yes, this chain will be shorter, so
    // ignore
    if currentBlock.PartOfChain {
      continue
    }

    count := 1 //including "genesis"
    for !bytes.Equal(currentBlock.PrevHash[0:32], options.StopAtPrevHash[0:32]) {
      oldBlock := currentBlock
      currentBlock = blockMap[currentBlock.PrevHash]
      if currentBlock == nil {
        break
      }
      if currentBlock.PartOfChain {
        break
      }
      oldBlock.PrevBlockInfo = currentBlock
      count++
    }
    if currentBlock != nil && count > 1 {

      chain := new(Chain)
      chain.Index = i
      chain.Last = blockOrder[i]
      chain.Length = count

      bi := chain.Last
      bi.PartOfChain = true

      for !bytes.Equal(bi.PrevHash[0:32], options.StopAtPrevHash[0:32]) {
        bi = bi.PrevBlockInfo
        bi.PartOfChain = true
      }

      chains = append(chains, chain)
    }

  }

  fmt.Printf("found %d possible chains\n", len(chains))

  sort.Slice(chains, func(i, j int) bool {
    return chains[i].Length > chains[j].Length
  })

  for i := 0; i < len(chains); i++ {
    chains[i].walkBack(options.StopAtPrevHash)
  }

  return chains, nil

}

func (bc *BitcoinBlockchainParser) parseBlockInfo(file *os.File) (*BlockInfo, error) {

  blockInfo := new(BlockInfo)
  var err error
  var skipped int

  // Read first 4 bytes of blockdata
  skipped, err = file.Read(buffer4)
  if err != nil || skipped != 4 {
    fmt.Println("Skip")
    return nil, err
  }

  // Size
  skipped, err = file.Read(buffer4)
  if err != nil {
    fmt.Println("Read size")
    return nil, err
  }
  blockInfo.Size = binary.LittleEndian.Uint32(buffer4)
  if blockInfo.Size == 0 {
    return nil, errors.New("Size is 0")
  }

  // Header
  /* Read next 80 bytes which will contain
    * version (4 bytes)
        * hash of previous blockInfo (32 bytes)
    * merkle root (32 bytes)
        * time stamp (4 bytes)
      * difficulty (4 bytes)
    * nonce (4 bytes)
  */
  skipped, err = file.Read(buffer80)
  if err != nil || skipped != 80 {
    fmt.Println("Read header")
    return nil, err
  }

  copy(blockInfo.PrevHash[:], buffer80[4:36])
  ReverseBytes(blockInfo.PrevHash[:])

  // Create blockInfo hash from those 80 bytes
  pass := sha256.Sum256(buffer80)
  copy(buffer32, pass[:])
  pass = sha256.Sum256(buffer32)
  copy(buffer32, pass[:])
  ReverseBytes(buffer32)
  copy(blockInfo.Hash[:], buffer32)

  return blockInfo, nil

}

func (bc *BitcoinBlockchainParser) ParseBlocks( chain *Chain, options *BitcoinBlockchainParserOptions) error {

  if chain == nil {
    return errors.New("No chain specified")
  }

  chain.walkBack( options.StopAtPrevHash )

  var err error
  blockInfo := chain.First

  // blockInfo os now genesis: walk forward and parse blocks
  var fileName string
  var file *os.File
  blockCount := 0
  start := time.Now()

  for blockInfo != nil {
    // read from blk file

    if options.CallBlockInfoCallback {
      if bc.onBlock != nil {
        err := bc.onBlockInfo(blockCount, chain.Length, blockInfo)
        if err != nil {
          return err
        }
      }
    }

    if options.CallBlockCallback {
      oldFileName := fileName
      oldFile := file

      fileName = path.Join(bc.directory, fmt.Sprintf("blk%.5d.dat", blockInfo.BlkFileNumber))

      if oldFileName != fileName {

        if oldFile != nil {
          oldFile.Close()
        }

        file, err = os.Open(fileName)
        if err != nil {
          return err
        }
      }

      // seek to position in file and parse Block from there
      _, err = file.Seek(int64(blockInfo.BlkFilePosition), 0)
      if err != nil {
        return err
      }
      block, bytesUsed, err := bc.parseBlock(file)
      if err != nil {
        return err
      }
      if block == nil {
        break
      }
      if int(block.Size) != bytesUsed-8 {
        return errors.New("Data mismatch")
      }

      if bc.onBlock != nil {

        err = bc.onBlock(blockCount, chain.Length, block)
        if err != nil {
          return err
        }
      }
    }

    if chain.Length-blockCount < 0 {
      fmt.Println("Something is wrong")
    }

    if blockCount != 0 && blockCount%1000 == 0 {
      elapsed := time.Since(start)
      nanosPerBlock := elapsed.Nanoseconds() / int64(blockCount)
      fmt.Printf("Traversing %d blocks took: %s\n", blockCount, elapsed)
      fmt.Printf("Traversing 1 blockInfo took: %s\n", time.Duration(nanosPerBlock))
      fmt.Printf("Number of blocks visited: %d\n", blockCount)
      fmt.Printf("Done: %3.2f percent\n", float64(blockCount)*100.0/float64(chain.Length))
      fmt.Printf("ETA: %s\n", time.Duration(nanosPerBlock*int64(chain.Length-blockCount)))
      fmt.Println("---")

    }
    blockCount++
    // next one
    blockInfo = blockInfo.NextBlockInfo
  }

  elapsed := time.Since(start)
  fmt.Printf("Traversing %d blocks took: %s\n", blockCount, elapsed)
  fmt.Println("---")
  return nil
}

func (bc *BitcoinBlockchainParser) parseBlock(file *os.File) (*Block, int, error) {

  block := new(Block)
  bytesUsed := 0
  var err error
  var skipped int

  // Skip first 4 bytes of blockdata
  // TODO: What is this?! Maybe some block marker
  _, err = file.Seek(4, 1)
  if err != nil {
    fmt.Println("Skip")
    return nil, 0, err
  }
  bytesUsed += 4
  POSITION_IN_FILE += 4

  // Size
  skipped, err = file.Read(buffer4)
  if err != nil {
    fmt.Println("Read size")
    return nil, 0, err
  }
  block.Size = binary.LittleEndian.Uint32(buffer4)
  bytesUsed += skipped
  POSITION_IN_FILE += skipped

  // Header
  /* Read next 80 bytes which will contain
    * version (4 bytes)
        * hash of previous block (32 bytes)
    * merkle root (32 bytes)
        * time stamp (4 bytes)
      * difficulty (4 bytes)
    * nonce (4 bytes)
  */
  skipped, err = file.Read(buffer80)
  if err != nil || skipped != 80 {
    fmt.Println("Read header")
    return nil, 0, err
  }
  bytesUsed += skipped
  POSITION_IN_FILE += skipped

  block.Version = binary.LittleEndian.Uint32(buffer80[0:4])
  copy(block.PrevHash[:], buffer80[4:36])

  ReverseBytes(block.PrevHash[:])

  copy(block.MerkleRoot[:], buffer80[36:68])
  block.Timestamp = binary.LittleEndian.Uint32(buffer80[68:72])
  copy(block.Difficulty[:], buffer80[72:76])
  block.Nonce = binary.LittleEndian.Uint32(buffer80[76:80])

  // Create block hash from those 80 bytes
  pass := sha256.Sum256(buffer80)
  copy(buffer32, pass[:])
  pass = sha256.Sum256(buffer32)
  copy(buffer32, pass[:])
  ReverseBytes(buffer32)
  copy(block.Hash[:], buffer32)

  // Transaction count bytes
  skipped, err = file.Read(buffer1)
  if err != nil || skipped != 1 {
    fmt.Println("Read tx count")
    return nil, 0, err
  }
  bytesUsed += skipped
  POSITION_IN_FILE += skipped

  txCount, txCountBytesUsed, _, err := readCount(buffer1[0], file)
  POSITION_IN_FILE += txCountBytesUsed

  if err != nil {
    fmt.Println("Read tx count")
    return nil, 0, err
  }

  bytesUsed += txCountBytesUsed

  if txCount > 0 {
    transactions, txBytesUsed, err := parseTransactions(file, int(txCount))

    if err != nil {
      fmt.Println("Read txs")
      return nil, 0, err
    }

    bytesUsed += txBytesUsed
    block.Transactions = transactions
  }

  return block, bytesUsed, nil

}

func parseTransactions(file *os.File, transactionCount int) ([]Transaction, int, error) {
  transactions := make([]Transaction, transactionCount)

  bytesUsed := 0
  var err error
  var skipped int

  for t := 0; t < transactionCount; t++ {
    txidData := make([]byte, 0)
    wtxidData := make([]byte, 0)
    // Version
    txSize := 0
    txBaseSize := 0
    skipped, err = file.Read(buffer4)
    if err != nil || skipped != 4 {
      fmt.Println("Read version")
      return nil, 0, err
    }
    bytesUsed += skipped
    POSITION_IN_FILE += skipped
    txSize += skipped
    txBaseSize += skipped

    transactions[t].Version = binary.LittleEndian.Uint32(buffer4)

    txidData = append(txidData, buffer4...)
    wtxidData = append(wtxidData, buffer4...)

    skipped, err = file.Read(buffer1)
    if err != nil || skipped != 1 {
      fmt.Println("Read input count 0")
      return nil, 0, err
    }
    bytesUsed += skipped
    POSITION_IN_FILE += skipped
    txSize += skipped
    txBaseSize += skipped

    b := buffer1[0]

    // is witness flag present?
    // 0 says yes, cause there are no tx with 0 inputs
    if b == 0 {
      skipped, err = file.Read(buffer2)
      if err != nil || skipped != 2 {
        fmt.Println("Read input count 1")
        return nil, 0, err
      }
      bytesUsed += skipped
      POSITION_IN_FILE += skipped
      txSize += skipped
      txBaseSize += skipped

      b = buffer2[1]
      transactions[t].Witness = true
    }

    txidData = append(txidData, b)
    wtxidData = append(wtxidData, b)
    inputCount, inputCountBytesUsed, rawBytes, err := readCount(b, file)
    bytesUsed += inputCountBytesUsed
    POSITION_IN_FILE += inputCountBytesUsed
    txSize += inputCountBytesUsed
    txBaseSize += inputCountBytesUsed

    // TODO: the following might be wrong :)
    txidData = append(txidData, rawBytes...)
    wtxidData = append(wtxidData, rawBytes...)

    if err != nil {
      fmt.Println("input count")
      return nil, 0, err
    }

    transactions[t].Inputs = make([]TxInput, inputCount)

    for i := 0; i < int(inputCount); i++ {

      // Source tx hash
      skipped, err = file.Read(buffer32)
      if err != nil || skipped != 32 {
        fmt.Println("Read input hash")
        return nil, 0, err
      }
      bytesUsed += skipped
      POSITION_IN_FILE += skipped
      txSize += skipped
      txBaseSize += skipped

      copy(transactions[t].Inputs[i].SourceTxHash[:], buffer32)

      txidData = append(txidData, buffer32...)
      wtxidData = append(wtxidData, buffer32...)

      // Source tx output index
      skipped, err = file.Read(buffer4)
      if err != nil || skipped != 4 {
        fmt.Println("Read input index")
        return nil, 0, err
      }
      bytesUsed += skipped
      POSITION_IN_FILE += skipped
      txSize += skipped
      txBaseSize += skipped

      transactions[t].Inputs[i].OutputIndex = binary.LittleEndian.Uint32(buffer4)

      txidData = append(txidData, buffer4...)
      wtxidData = append(wtxidData, buffer4...)

      // Script length
      skipped, err = file.Read(buffer1)
      if err != nil || skipped != 1 {
        fmt.Println("Read script length")
        return nil, 0, err
      }
      bytesUsed += skipped
      POSITION_IN_FILE += skipped
      txSize += skipped
      txBaseSize += skipped

      txidData = append(txidData, buffer1...)
      wtxidData = append(wtxidData, buffer1...)

      scriptLength, scriptLengthBytesUsed, rawBytes, err := readCount(buffer1[0], file)
      bytesUsed += scriptLengthBytesUsed
      POSITION_IN_FILE += scriptLengthBytesUsed
      txSize += scriptLengthBytesUsed
      txBaseSize += scriptLengthBytesUsed

      // TODO: prolly broken
      txidData = append(txidData, rawBytes...)
      wtxidData = append(wtxidData, rawBytes...)

      // Script
      if scriptLength > 0 {

        tmpBuffer := make([]byte, scriptLength)

        skipped, err = file.Read(tmpBuffer)
        if err != nil || skipped != int(scriptLength) {
          fmt.Println("Read input script", err)
          return nil, 0, err
        }
        bytesUsed += skipped
        POSITION_IN_FILE += skipped
        txSize += skipped
        txBaseSize += skipped

        transactions[t].Inputs[i].Script = tmpBuffer

        // TODO: prolly broken
        txidData = append(txidData, tmpBuffer...)
        wtxidData = append(wtxidData, tmpBuffer...)
      }

      // Sequence
      skipped, err = file.Read(buffer4)
      if err != nil || skipped != 4 {
        fmt.Println("Read sequence")
        return nil, 0, err
      }
      bytesUsed += skipped
      POSITION_IN_FILE += skipped
      txSize += skipped
      txBaseSize += skipped

      transactions[t].Inputs[i].Sequence = binary.LittleEndian.Uint32(buffer4)

      // TODO: prolly broken
      txidData = append(txidData, buffer4...)
      wtxidData = append(wtxidData, buffer4...)
    }

    // Output count
    skipped, err = file.Read(buffer1)
    if err != nil || skipped != 1 {
      fmt.Println("Peek output count")
      return nil, 0, err
    }
    bytesUsed += skipped
    POSITION_IN_FILE += skipped
    txSize += skipped
    txBaseSize += skipped

    txidData = append(txidData, buffer1...)
    wtxidData = append(wtxidData, buffer1...)

    outputCount, outputCountBytesUsed, rawBytes, err := readCount(buffer1[0], file)
    if err != nil {
      fmt.Println("output count")
      return nil, 0, err
    }

    bytesUsed += outputCountBytesUsed
    POSITION_IN_FILE += outputCountBytesUsed
    txSize += outputCountBytesUsed
    txBaseSize += outputCountBytesUsed

    // TODO: the following might be wrong :)
    txidData = append(txidData, rawBytes...)
    wtxidData = append(wtxidData, rawBytes...)

    transactions[t].Outputs = make([]TxOutput, outputCount)

    for o := 0; o < int(outputCount); o++ {

      // Value
      skipped, err = file.Read(buffer8)
      if err != nil || skipped != 8 {
        fmt.Println("Read value")
        return nil, 0, err
      }
      bytesUsed += skipped
      POSITION_IN_FILE += skipped
      txSize += skipped
      txBaseSize += skipped

      transactions[t].Outputs[o].Value = binary.LittleEndian.Uint64(buffer8)

      txidData = append(txidData, buffer8...)
      wtxidData = append(wtxidData, buffer8...)

      // Script length
      skipped, err = file.Read(buffer1)
      if err != nil || skipped != 1 {
        fmt.Println("Read script length")
        return nil, 0, err
      }
      bytesUsed += skipped
      POSITION_IN_FILE += skipped
      txSize += skipped
      txBaseSize += skipped

      txidData = append(txidData, buffer1...)
      wtxidData = append(wtxidData, buffer1...)

      scriptLength, scriptLengthBytesUsed, rawBytes, err := readCount(buffer1[0], file)
      if err != nil {
        fmt.Println("script length")
        return nil, 0, err
      }

      bytesUsed += scriptLengthBytesUsed
      POSITION_IN_FILE += scriptLengthBytesUsed
      txSize += scriptLengthBytesUsed
      txBaseSize += scriptLengthBytesUsed

      txidData = append(txidData, rawBytes...)
      wtxidData = append(wtxidData, rawBytes...)

      // Script
      if scriptLength > 0 {
        tmpBuffer := make([]byte, scriptLength)

        skipped, err = file.Read(tmpBuffer)
        if err != nil || skipped != int(scriptLength) {
          fmt.Println("Read output script", err)
          return nil, 0, err
        }
        bytesUsed += skipped
        POSITION_IN_FILE += skipped
        txSize += skipped
        txBaseSize += skipped

        transactions[t].Outputs[o].Script = NewScript(tmpBuffer)
        txidData = append(txidData, tmpBuffer...)
        wtxidData = append(wtxidData, tmpBuffer...)
      }
    }

    if transactions[t].Witness {
      // Witness length
      for i := 0; i < int(inputCount); i++ {
        skipped, err = file.Read(buffer1)
        if err != nil || skipped != 1 {
          fmt.Println("Read witness length")
          return nil, 0, err
        }
        bytesUsed += skipped
        POSITION_IN_FILE += skipped
        txSize += skipped

        wtxidData = append(wtxidData, buffer1...)

        witnessLength, witnessLengthBytesUsed, rawBytes, err := readCount(buffer1[0], file)
        if err != nil {
          fmt.Println("witness length")
          return nil, 0, err
        }

        bytesUsed += witnessLengthBytesUsed
        POSITION_IN_FILE += witnessLengthBytesUsed
        txSize += witnessLengthBytesUsed

        wtxidData = append(wtxidData, rawBytes...)

        // Witness
        transactions[t].WitnessItems = make([]WitnessItem, witnessLength)
        for w := 0; w < int(witnessLength); w++ {
          // Witness item length
          skipped, err = file.Read(buffer1)
          if err != nil || skipped != 1 {
            fmt.Println("Read witness item length")
            return nil, 0, err
          }
          bytesUsed += skipped
          POSITION_IN_FILE += skipped
          txSize += skipped

          wtxidData = append(wtxidData, buffer1...)

          witnessItemLength, witnessItemLengthBytesUsed, rawBytes, err := readCount(buffer1[0], file)
          if err != nil {
            fmt.Println("witness item length")
            return nil, 0, err
          }

          bytesUsed += witnessItemLengthBytesUsed
          POSITION_IN_FILE += witnessItemLengthBytesUsed
          txSize += witnessItemLengthBytesUsed

          wtxidData = append(wtxidData, rawBytes...)

          tmpBuffer := make([]byte, witnessItemLength)

          //skipped64, err := file.Seek(int64(witnessItemLength),1)
          witnessPosition := POSITION_IN_FILE
          skipped, err = file.Read(tmpBuffer)
          if err != nil || skipped != int(witnessItemLength) {
            fmt.Println("Read witness")
            return nil, 0, err
          }
          transactions[t].WitnessItems[w].Data = tmpBuffer
          transactions[t].WitnessItems[w].BlkFilePosition = witnessPosition
          bytesUsed += skipped
          POSITION_IN_FILE += skipped
          txSize += skipped

          wtxidData = append(wtxidData, tmpBuffer...)

        }
      }
    }

    // Lock time
    skipped, err = file.Read(buffer4)
    if err != nil || skipped != 4 {
      fmt.Println("Read lock time")
      return nil, 0, err
    }
    bytesUsed += skipped
    POSITION_IN_FILE += skipped
    txSize += skipped
    txBaseSize += skipped

    transactions[t].Locktime = binary.LittleEndian.Uint32(buffer4)
    transactions[t].Size = txSize
    transactions[t].BaseSize = txBaseSize
    transactions[t].Weight = txBaseSize*3 + txSize
    transactions[t].VirtualSize = int(math.Ceil(float64(transactions[t].Weight) / 4))

    txidData = append(txidData, buffer4...)
    wtxidData = append(wtxidData, buffer4...)

    // create txid
    pass := sha256.Sum256(txidData)
    copy(buffer32, pass[:])
    pass = sha256.Sum256(buffer32)
    copy(buffer32, pass[:])
    ReverseBytes(buffer32)
    copy(transactions[t].TxId[:], buffer32)

    // create wtxid
    pass = sha256.Sum256(wtxidData)
    copy(buffer32, pass[:])
    pass = sha256.Sum256(buffer32)
    copy(buffer32, pass[:])
    ReverseBytes(buffer32)
    copy(transactions[t].WtxId[:], buffer32)

  }

  return transactions, bytesUsed, nil
}

func readCount(b byte, file *os.File) (uint64, int, []byte, error) {
  bytesUsed := int(0)

  val := uint64(0)
  var rawBytes []byte

  if b < 253 {
    val = uint64(b)
    rawBytes = []byte{}
  } else {
    byteCount := 0

    if b == 253 {
      byteCount = 2
    } else if b == 254 {
      byteCount = 4
    } else if b == 255 {
      byteCount = 8
    }

    var bytes []byte
    if byteCount == 2 {
      bytes = buffer2
    } else if byteCount == 4 {
      bytes = buffer4
    } else if byteCount == 8 {
      bytes = buffer8
    }

    skipped, err := file.Read(bytes)
    if err != nil || skipped != byteCount {
      fmt.Println("Read count 1")
      return 0, 0, rawBytes, err
    }
    bytesUsed += skipped

    rawBytes = bytes
    copy(buffer8[0:byteCount], bytes)
    for i := 0; i < 8-byteCount; i++ {
      buffer8[i+byteCount] = 0
    }

    val = binary.LittleEndian.Uint64(buffer8)
  }
  return val, bytesUsed, rawBytes, nil
}

func ReverseBytes(bytes []byte) {
  for i, j := 0, len(bytes)-1; i < j; i, j = i+1, j-1 {
    bytes[i], bytes[j] = bytes[j], bytes[i]
  }
}
