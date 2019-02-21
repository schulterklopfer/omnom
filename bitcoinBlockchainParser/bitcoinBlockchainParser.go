package bitcoinBlockchainParser

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/pkg/errors"
	"io/ioutil"
	"log"
	"math"
	"os"
	"path"
	"sort"
	"strings"
	"time"
)

type BitcoinBlockchainParser struct {
	// private
	directory string
	onBlock OnBlockCallback
}

type OnBlockCallback func( int, int, *Block )

type BlockIndex struct {
	Hash [32]byte
	Size uint32
	PrevHash [32]byte
	PrevBlockIndex *BlockIndex
	NextBlockIndex *BlockIndex
	BlkFileIndex uint16
	Position int64
	Index uint64
	PartOfChain bool
}

type Block struct {
	Hash [32]byte
	Size uint32
	// Header
	Version uint32
	PrevHash [32]byte
	MerkleRoot [32]byte
	Timestamp uint32
	Difficulty [4]byte
	Nonce uint32

	//Transactions
	Transactions []Transaction

	HashString string
}

type Chain struct {
	Index int
	Tip *BlockIndex
	Length int
}

type Transaction struct {
	TxId [32]byte
	TxIdString string
	WtxId [32]byte
	WtxIdString string
	Version uint32
	Witness bool
	Size int
	BaseSize int
	VirtualSize int
	Weight int
	Amount uint64
	Fee uint64
	Inputs []TxInput
	Outputs []TxOutput
	WitnessItems []WitnessItem
	Locktime uint32

	//TxIdString string
	//WtxIdString string

}

type WitnessItem struct {
	Data []byte
}

type TxInput struct {
	SourceTxHash [32]byte
	SourceTxHashString string
	OutputIndex uint32
	Script []byte
	Sequence uint32
}

type TxOutput struct {
	Value uint64
	Script *Script

}

func NewBitcoinBlockchainParser( directory string,  onBlock OnBlockCallback ) *BitcoinBlockchainParser {
	return &BitcoinBlockchainParser{directory, onBlock }
}

func allZero( hash [32]byte ) bool {
	for i:=0; i<32; i++ {
		if hash[i] != 0x00 {
			return false
		}
	}
	return true
}

func (bi *BlockIndex) isGenesis() bool {
	return !bi.hasPrev()
}


func (bi *BlockIndex) hasPrev() bool {
	return !allZero(bi.PrevHash)
}

func (bi *BlockIndex) isPrevTo( blockIndex *BlockIndex ) bool {
	if bi == nil || blockIndex == nil {
		return false
	}
	for i:=0; i<32; i++ {
		if bi.Hash[i] != blockIndex.PrevHash[i] {
			return false
		}
	}
	return true
}

func (bi *BlockIndex) isEqualTo( blockIndex *BlockIndex ) bool {
	if bi == nil || blockIndex == nil {
		return false
	}
	for i:=0; i<32; i++ {
		if bi.Hash[i] != blockIndex.Hash[i] {
			return false
		}
	}
	return true
}

func (bc *BitcoinBlockchainParser ) Close() {

}

func filterBlockDataFiles(fileInfos []os.FileInfo) (ret []os.FileInfo) {
	for i:=0; i<len(fileInfos); i++ {
		if strings.HasPrefix(fileInfos[i].Name(), "blk") && strings.HasSuffix(fileInfos[i].Name(),".dat") {
			ret = append(ret, fileInfos[i])
		}
	}
	return
}

var POSITION_IN_FILE int
var CHAINCFG = &chaincfg.TestNet3Params

// todo: use standard length buffers for 4,8,32 and only alloc for variable lengths exceeding 4096 bytes
var buffer1 = make([]byte,1)
var buffer2 = make([]byte,2)
var buffer4 = make([]byte,4)
var buffer8 = make([]byte,8)
var buffer32 = make([]byte,32)
var buffer80 = make([]byte,80)
var buffer4096 = make([]byte,4096)

func (bc *BitcoinBlockchainParser ) buildBlockIndexChains() ([]*Chain, error) {
	fileInfos, err := ioutil.ReadDir(bc.directory)
	if err != nil {
		return nil, err
	}

	fileInfos = filterBlockDataFiles(fileInfos)
	start := time.Now()
	blockCount := 0

	blockIndexOrder := make( []*BlockIndex, 0 )
	blockIndexMap := make( map[[32]byte]*BlockIndex )

	for index, fileInfo := range fileInfos {
		fmt.Printf("Opening %s [%d of %d]\n", fileInfo.Name(), index+1, len(fileInfos) )

		// Open readonly
		file, err := os.Open( path.Join(bc.directory, fileInfo.Name()))
		if err != nil {
			return nil, err
		}
		POSITION_IN_FILE = 0
		//fmt.Println(err, reader.Size(), fileInfo.Size() )
		nextBlockIndexPosition := int64(0)

		for nextBlockIndexPosition < fileInfo.Size() {
			blockIndex, err := bc.parseBlockIndex(file)
			if blockIndex == nil {
				break
			}

			fmt.Sscanf( fileInfo.Name(), "blk%d.dat", &blockIndex.BlkFileIndex )
			blockIndex.Position = nextBlockIndexPosition
			blockIndex.Index = uint64(blockCount)

			blockIndexOrder = append( blockIndexOrder, blockIndex )
			blockIndexMap[blockIndex.Hash] = blockIndex

			nextBlockIndexPosition += int64(blockIndex.Size)+8
			_, err = file.Seek(nextBlockIndexPosition,0)

			if err != nil {
				break
			}

			blockCount++

		}
		elapsed := time.Since(start)
		fmt.Printf("Traversing 1 block took: %s\n", time.Duration(elapsed.Nanoseconds()/int64(blockCount)))
		fmt.Printf("Traversing %d blocks took: %s\n", blockCount, elapsed)
		fmt.Printf("Done: %d of %d bytes, %d blocks \n", nextBlockIndexPosition, fileInfo.Size(), blockCount )
		fmt.Println("---")

		file.Close()
	}
	elapsed := time.Since(start)
	fmt.Printf("Traversing %d blocks took: %s\n", blockCount, elapsed )

	fmt.Println("\nlooking for chain")

	// make genesis block
	// TODO remove
	for i:=0; i<32; i++  {
		blockIndexOrder[0].PrevHash[i]=0x00
	}


	chains := make( []*Chain,0 )

	for i:=len( blockIndexOrder)-1; i>=0; i-- {
		currentBlockIndex := blockIndexOrder[i]

		// is this block part of another chain?
		// if yes, this chain will be shorter, so
		// ignore
		if currentBlockIndex.PartOfChain {
			continue
		}

		count := 0
		for !currentBlockIndex.isGenesis() {
			oldBlockIndex := currentBlockIndex
			currentBlockIndex = blockIndexMap[currentBlockIndex.PrevHash]
			if currentBlockIndex == nil {
				break
			}
			if currentBlockIndex.PartOfChain {
				break
			}
			oldBlockIndex.PrevBlockIndex = currentBlockIndex
			count++
		}
		if currentBlockIndex != nil && count > 0 {

			chain := new(Chain)
			chain.Index = i
			chain.Tip = blockIndexOrder[i]
			chain.Length = count

			bi := chain.Tip
			bi.PartOfChain = true

			for !bi.isGenesis() {
				bi = bi.PrevBlockIndex
				bi.PartOfChain = true
			}

			chains = append( chains, chain )
		}

	}

	fmt.Printf("found %d possible chains\n", len(chains) )

	sort.Slice(chains, func(i, j int) bool {
		return chains[i].Length > chains[j].Length
	})

	return chains, nil

}

func (bc *BitcoinBlockchainParser) parseBlockIndex( file *os.File ) (*BlockIndex, error) {

	blockIndex := new(BlockIndex)
	var err error
	var skipped int

	// Read first 4 bytes of blockdata
	skipped, err = file.Read(buffer4)
	if err != nil || skipped != 4 {
		fmt.Println("Skip")
		return nil,err
	}

	// Size
	skipped,err = file.Read(buffer4)
	if err != nil {
		fmt.Println("Read size")
		return nil,err
	}
	blockIndex.Size = binary.LittleEndian.Uint32(buffer4)
	if blockIndex.Size == 0 {
		return nil, errors.New("Size is 0")
	}

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
		return nil,err
	}

	copy(blockIndex.PrevHash[:], buffer80[4:36])
	ReverseBytes(blockIndex.PrevHash[:])

	// Create block hash from those 80 bytes
	pass := sha256.Sum256(buffer80)
	copy( buffer32, pass[:] )
	pass = sha256.Sum256( buffer32 )
	copy( buffer32, pass[:] )
	ReverseBytes(buffer32)
	copy( blockIndex.Hash[:], buffer32 )

	return blockIndex,nil

}

func (bc *BitcoinBlockchainParser ) ParseBlocks() {

	chains, err := bc.buildBlockIndexChains()
	if err != nil {
		log.Fatal(err)
		return
	}

	// chains is sorted by length
	longestChain := chains[0]

	bi := longestChain.Tip

	for !bi.isGenesis() {
		oldBi := bi
		bi = bi.PrevBlockIndex
		bi.NextBlockIndex = oldBi
	}

	// bi os now genesis: walk forward and parse blocks
	var fileName string
	var file *os.File
	blockCount := 0
	start := time.Now()

	for bi.NextBlockIndex != nil {
		// read from blk file
		oldFileName := fileName
		oldFile := file

		fileName = path.Join( bc.directory, fmt.Sprintf( "blk%.5d.dat", bi.BlkFileIndex ) )

		if oldFileName != fileName {

			if oldFile != nil {
				oldFile.Close()
			}

			file, err = os.Open( fileName )
			if err != nil {
				log.Fatal(err)
				return
			}
		}

		// seek to position in file and parse Block from there
		_, err = file.Seek(bi.Position,0)

		block, bytesUsed, err := bc.parseBlock(file)
		if block == nil {
			fmt.Println(err)
			break
		}
		if int(block.Size) != bytesUsed-8 {
			fmt.Println("Data mismatch")
			break
		}

		if bc.onBlock != nil {
			bc.onBlock( blockCount, longestChain.Length, block )
		}

		if err != nil {
			break
		}

		if  blockCount != 0 && blockCount%1000 == 0 {
			elapsed := time.Since(start)
			fmt.Printf("Traversing %d blocks took: %s\n", blockCount, elapsed)
			fmt.Printf("Traversing 1 block took: %s\n", time.Duration(elapsed.Nanoseconds()/int64(blockCount)))
			fmt.Printf("Number of blocks visited: %d\n", blockCount)
			fmt.Printf("Done: %3.2f percent\n", float64(blockCount)*100.0/float64(longestChain.Length) )
			fmt.Println("---")

		}
		blockCount++

		// next one
		bi = bi.NextBlockIndex
	}

	elapsed := time.Since(start)
	fmt.Printf("Traversing %d blocks took: %s\n", blockCount, elapsed)
	fmt.Println("---")



	/*
	blockCount := 0

	for index, fileInfo := range fileInfos {
		fmt.Printf("Opening %s [%d of %d]\n", fileInfo.Name(), index+1, len(fileInfos) )

		// Open readonly
		file, err := os.Open( path.Join(bc.directory, fileInfo.Name()))
		if err != nil {
			log.Fatal(err)
			return
		}
		POSITION_IN_FILE = 0
		//fmt.Println(err, reader.Size(), fileInfo.Size() )
		nextBlockPosition := int64(0)

		for nextBlockPosition < fileInfo.Size() {
			block, bytesUsed, err := bc.parseBlock(file)
			if block == nil {
				fmt.Println(err)
				break
			}
			if int(block.Size) != bytesUsed-8 {
				fmt.Println("Data mismatch")
				break
			}

			if bc.onBlock != nil {
				bc.onBlock( blockCount, block )
			}

			nextBlockPosition += int64(block.Size)+8
			_, err = file.Seek(nextBlockPosition,0)

			if err != nil {
				break
			}

			if  blockCount != 0 && blockCount%1000 == 0 {
				elapsed := time.Since(start)
				fmt.Printf("Traversing %d blocks took: %s\n", blockCount, elapsed)
				fmt.Printf("Traversing 1 block took: %s\n", time.Duration(elapsed.Nanoseconds()/int64(blockCount)))
				fmt.Printf("Number of blocks visited: %d\n", blockCount)
				fmt.Printf("Bytes processsed: %d of %d \n", nextBlockPosition, fileInfo.Size() )
				fmt.Printf("%x %d\n", block.Hash, len(block.Transactions) )
				fmt.Println("---")

			}
			blockCount++

		}
		fmt.Printf("Done: %d of %d bytes, %d blocks \n", nextBlockPosition, fileInfo.Size(), blockCount )
		fmt.Println("---")

		file.Close()
	}
	fmt.Printf("Traversing all blocks took: %s\n", blockCount, time.Since(start))
	*/
}

func (bc *BitcoinBlockchainParser) parseBlock( file *os.File ) (*Block, int, error) {

	block := new(Block)
	bytesUsed := 0
	var err error
	var skipped int

	// Read first 4 bytes of blockdata
	// TODO: What is this?! Maybe some block marker
	skipped, err = file.Read(buffer4)
	if err != nil || skipped != 4 {
		fmt.Println("Skip")
		return nil,0,err
	}
	bytesUsed += skipped
	POSITION_IN_FILE +=skipped

	// Size
	skipped,err = file.Read(buffer4)
	if err != nil {
		fmt.Println("Read size")
		return nil,0,err
	}
	block.Size = binary.LittleEndian.Uint32(buffer4)
	bytesUsed += skipped
	POSITION_IN_FILE +=skipped


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
		return nil,0,err
	}
	bytesUsed += skipped
	POSITION_IN_FILE +=skipped

	block.Version = binary.LittleEndian.Uint32(buffer80[0:4])
	copy(block.PrevHash[:], buffer80[4:36])

	ReverseBytes(block.PrevHash[:])

	copy(block.MerkleRoot[:], buffer80[36:68])
	block.Timestamp = binary.LittleEndian.Uint32(buffer80[68:72])
	copy(block.Difficulty[:], buffer80[72:76])
	block.Nonce = binary.LittleEndian.Uint32(buffer80[76:80])

	// Create block hash from those 80 bytes
	pass := sha256.Sum256(buffer80)
	copy( buffer32, pass[:] )
	pass = sha256.Sum256( buffer32 )
	copy( buffer32, pass[:] )
	ReverseBytes(buffer32)
	copy( block.Hash[:], buffer32 )

	block.HashString = fmt.Sprintf("%x", block.Hash )

	// Transaction count bytes
	skipped, err = file.Read(buffer1)
	if err != nil || skipped != 1 {
		fmt.Println("Read tx count")
		return nil,0,err
	}
	bytesUsed+=skipped
	POSITION_IN_FILE +=skipped

	txCount, txCountBytesUsed, _, err := readCount(buffer1[0], file)
	POSITION_IN_FILE +=txCountBytesUsed

	if err != nil {
		fmt.Println("Read tx count")
		return nil,0,err
	}

	bytesUsed += txCountBytesUsed


	if txCount > 0 {
		transactions, txBytesUsed, err := parseTransactions( file, int(txCount) )

		if err != nil {
			fmt.Println("Read txs")
			return nil,0,err
		}

		bytesUsed += txBytesUsed
		block.Transactions = transactions
	}


	return block,bytesUsed,nil

}

func parseTransactions( file *os.File, transactionCount int ) ([]Transaction,int,error) {
	transactions := make( []Transaction, transactionCount )

	bytesUsed := 0
	var err error
	var skipped int
	var tmpBuffer []byte

	for t:=0; t<transactionCount; t++ {
		txidData := make([]byte,0)
		wtxidData := make([]byte,0)
		// Version
		txSize := 0
		txBaseSize := 0
		skipped, err = file.Read(buffer4 )
		if err != nil || skipped != 4 {
			fmt.Println("Read version")
			return nil,0,err
		}
		bytesUsed += skipped
		POSITION_IN_FILE +=skipped
		txSize += skipped
		txBaseSize += skipped

		transactions[t].Version = binary.LittleEndian.Uint32(buffer4)

		txidData = append( txidData, buffer4... )
		wtxidData = append( wtxidData, buffer4... )

		skipped, err = file.Read(buffer1)
		if err != nil || skipped != 1 {
			fmt.Println("Read input count 0")
			return nil,0,err
		}
		bytesUsed += skipped
		POSITION_IN_FILE +=skipped
		txSize += skipped
		txBaseSize += skipped

		txidData = append( txidData, buffer1... )
		wtxidData = append( wtxidData, buffer1... )

		b := buffer1[0]

		// is witness flag present?
		// 0 says yes, cause there are no tx with 0 inputs
		if b==0 {
			skipped, err = file.Read(buffer2)
			if err != nil || skipped != 2 {
				fmt.Println("Read input count 1")
				return nil,0,err
			}
			bytesUsed += skipped
			POSITION_IN_FILE +=skipped
			txSize += skipped
			txBaseSize += skipped

			b = buffer2[1]

			txidData = append( txidData, buffer2... )
			wtxidData = append( wtxidData, buffer2... )

			transactions[t].Witness = true
		}

		inputCount,inputCountBytesUsed,rawBytes,err := readCount(b,file)
		bytesUsed+=inputCountBytesUsed
		POSITION_IN_FILE +=inputCountBytesUsed
		txSize += inputCountBytesUsed
		txBaseSize += inputCountBytesUsed

		// TODO: the following might be wrong :)
		txidData = append( txidData, rawBytes... )
		wtxidData = append( wtxidData, rawBytes... )

		if err != nil {
			fmt.Println("input count")
			return nil,0,err
		}

		transactions[t].Inputs = make([]TxInput,inputCount)

		for i:=0; i<int(inputCount); i++ {

			// Source tx hash
			skipped, err = file.Read(buffer32 )
			if err != nil || skipped != 32 {
				fmt.Println("Read input hash")
				return nil,0,err
			}
			bytesUsed += skipped
			POSITION_IN_FILE +=skipped
			txSize += skipped
			txBaseSize += skipped

			copy(transactions[t].Inputs[i].SourceTxHash[:],buffer32)
			transactions[t].Inputs[i].SourceTxHashString = fmt.Sprintf("%x",transactions[t].Inputs[i].SourceTxHash)

			txidData = append( txidData, buffer32... )
			wtxidData = append( wtxidData, buffer32... )


			// Source tx output index
			skipped, err = file.Read(buffer4)
			if err != nil || skipped != 4 {
				fmt.Println("Read input index")
				return nil,0,err
			}
			bytesUsed += skipped
			POSITION_IN_FILE +=skipped
			txSize += skipped
			txBaseSize += skipped

			transactions[t].Inputs[i].OutputIndex = binary.LittleEndian.Uint32(buffer4)

			txidData = append( txidData, buffer4... )
			wtxidData = append( wtxidData, buffer4... )

			// Script length
			skipped, err = file.Read(buffer1)
			if err != nil || skipped != 1 {
				fmt.Println("Read script length")
				return nil,0,err
			}
			bytesUsed += skipped
			POSITION_IN_FILE +=skipped
			txSize += skipped
			txBaseSize += skipped

			txidData = append( txidData, buffer1... )
			wtxidData = append( wtxidData, buffer1... )

			scriptLength,scriptLengthBytesUsed,rawBytes,err := readCount(buffer1[0], file)
			bytesUsed += scriptLengthBytesUsed
			POSITION_IN_FILE +=scriptLengthBytesUsed
			txSize += scriptLengthBytesUsed
			txBaseSize += scriptLengthBytesUsed

			// TODO: prolly broken
			txidData = append( txidData, rawBytes... )
			wtxidData = append( wtxidData, rawBytes... )


			// Script
			if scriptLength > 0 {

				if scriptLength > 4096 {
					tmpBuffer = make( []byte, scriptLength)
				} else {
					tmpBuffer = buffer4096[:scriptLength]
				}

				skipped, err = file.Read(tmpBuffer)
				if err != nil || skipped != int(scriptLength) {
					fmt.Println("Read input script", err)
					return nil,0,err
				}
				bytesUsed += skipped
				POSITION_IN_FILE +=skipped
				txSize += skipped
				txBaseSize += skipped

				transactions[t].Inputs[i].Script = tmpBuffer

				// TODO: prolly broken
				txidData = append( txidData, tmpBuffer... )
				wtxidData = append( wtxidData, tmpBuffer... )
			}

			// Sequence
			skipped, err = file.Read(buffer4)
			if err != nil || skipped != 4 {
				fmt.Println("Read sequence")
				return nil,0,err
			}
			bytesUsed += skipped
			POSITION_IN_FILE +=skipped
			txSize += skipped
			txBaseSize += skipped


			transactions[t].Inputs[i].Sequence = binary.LittleEndian.Uint32(buffer4)

			// TODO: prolly broken
			txidData = append( txidData, buffer4... )
			wtxidData = append( wtxidData, buffer4... )
		}

		// Output count
		skipped, err = file.Read(buffer1)
		if err != nil || skipped != 1 {
			fmt.Println("Peek output count")
			return nil,0,err
		}
		bytesUsed += skipped
		POSITION_IN_FILE +=skipped
		txSize += skipped
		txBaseSize += skipped

		txidData = append( txidData, buffer1... )
		wtxidData = append( wtxidData, buffer1... )

		outputCount,outputCountBytesUsed,rawBytes,err := readCount(buffer1[0], file)
		if err != nil {
			fmt.Println("output count")
			return nil,0,err
		}

		bytesUsed += outputCountBytesUsed
		POSITION_IN_FILE +=outputCountBytesUsed
		txSize += outputCountBytesUsed
		txBaseSize += outputCountBytesUsed

		// TODO: the following might be wrong :)
		txidData = append( txidData, rawBytes... )
		wtxidData = append( wtxidData, rawBytes... )


		transactions[t].Outputs = make([]TxOutput,outputCount)

		for o:=0; o<int(outputCount); o++ {

			// Value
			skipped, err = file.Read(buffer8)
			if err != nil || skipped != 8 {
				fmt.Println("Read value")
				return nil,0,err
			}
			bytesUsed += skipped
			POSITION_IN_FILE +=skipped
			txSize += skipped
			txBaseSize += skipped

			transactions[t].Outputs[o].Value = binary.LittleEndian.Uint64(buffer8)

			txidData = append( txidData, buffer8... )
			wtxidData = append( wtxidData, buffer8... )

			// Script length
			skipped, err = file.Read(buffer1)
			if err != nil || skipped != 1 {
				fmt.Println("Peek script length")
				return nil,0,err
			}
			bytesUsed += skipped
			POSITION_IN_FILE +=skipped
			txSize += skipped
			txBaseSize += skipped

			txidData = append( txidData, buffer1... )
			wtxidData = append( wtxidData, buffer1... )

			scriptLength,scriptLengthBytesUsed,rawBytes,err := readCount(buffer1[0], file)
			if err != nil {
				fmt.Println("script length")
				return nil,0,err
			}

			bytesUsed += scriptLengthBytesUsed
			POSITION_IN_FILE +=scriptLengthBytesUsed
			txSize += scriptLengthBytesUsed
			txBaseSize += scriptLengthBytesUsed

			txidData = append( txidData, rawBytes... )
			wtxidData = append( wtxidData, rawBytes... )

			// Script
			if scriptLength > 0 {
				if scriptLength > 4096 {
					tmpBuffer = make( []byte, scriptLength)
				} else {
					tmpBuffer = buffer4096[:scriptLength]
				}
				skipped, err = file.Read(tmpBuffer)
				if err != nil || skipped != int(scriptLength) {
					fmt.Println("Read output script", err)
					return nil,0,err
				}
				bytesUsed += skipped
				POSITION_IN_FILE +=skipped
				txSize += skipped
				txBaseSize += skipped

				transactions[t].Outputs[o].Script =  NewScript( tmpBuffer )

				txidData = append( txidData, tmpBuffer... )
				wtxidData = append( wtxidData, tmpBuffer... )
			}
		}

		if transactions[t].Witness {
			// Witness length
			for i:=0; i<int(inputCount); i++ {
				skipped, err = file.Read(buffer1)
				if err != nil || skipped != 1 {
					fmt.Println("Read witness length")
					return nil,0,err
				}
				bytesUsed += skipped
				POSITION_IN_FILE +=skipped
				txSize += skipped

				wtxidData = append( wtxidData, buffer1... )

				witnessLength,witnessLengthBytesUsed,rawBytes,err := readCount(buffer1[0], file)
				if err != nil {
					fmt.Println("witness length")
					return nil,0,err
				}

				bytesUsed += witnessLengthBytesUsed
				POSITION_IN_FILE +=witnessLengthBytesUsed
				txSize += witnessLengthBytesUsed

				wtxidData = append( wtxidData, rawBytes... )

				// Witness
				transactions[t].WitnessItems = make([]WitnessItem,witnessLength)
				for w:=0; w<int(witnessLength); w++ {
					// Witness item length
					skipped, err = file.Read(buffer1)
					if err != nil || skipped != 1 {
						fmt.Println("Read witness item length")
						return nil,0,err
					}
					bytesUsed += skipped
					POSITION_IN_FILE +=skipped
					txSize += skipped

					wtxidData = append( wtxidData, buffer1... )

					witnessItemLength,witnessItemLengthBytesUsed,rawBytes,err := readCount(buffer1[0], file)
					if err != nil {
						fmt.Println("witness item length")
						return nil,0,err
					}

					bytesUsed += witnessItemLengthBytesUsed
					POSITION_IN_FILE +=witnessItemLengthBytesUsed
					txSize += witnessItemLengthBytesUsed

					wtxidData = append( wtxidData, rawBytes... )

					if witnessItemLength > 4096 {
						tmpBuffer = make( []byte, witnessItemLength)
					} else {
						tmpBuffer = buffer4096[:witnessItemLength]
					}
					//skipped64, err := file.Seek(int64(witnessItemLength),1)
					skipped, err = file.Read(tmpBuffer)
					if err != nil || skipped != int(witnessItemLength) {
						fmt.Println("Read witness")
						return nil,0,err
					}
					transactions[t].WitnessItems[w].Data = tmpBuffer
					bytesUsed += skipped
					POSITION_IN_FILE +=skipped
					txSize += skipped

					wtxidData = append( wtxidData, tmpBuffer... )

				}
			}
		}

		// Lock time
		skipped, err = file.Read(buffer4)
		if err != nil || skipped != 4 {
			fmt.Println("Read lock time")
			return nil,0,err
		}
		bytesUsed += skipped
		POSITION_IN_FILE +=skipped
		txSize += skipped
		txBaseSize += skipped

		transactions[t].Locktime = binary.LittleEndian.Uint32(buffer4)
		transactions[t].Size = txSize
		transactions[t].BaseSize = txBaseSize
		transactions[t].Weight = txBaseSize * 3 + txSize
		transactions[t].VirtualSize = int(math.Ceil(float64(transactions[t].Weight)/4))

		txidData = append( txidData, buffer4... )
		wtxidData = append( wtxidData, buffer4... )

		// create txid
		pass := sha256.Sum256(txidData)
		copy( buffer32, pass[:] )
		pass = sha256.Sum256( buffer32 )
		copy( buffer32, pass[:] )
		ReverseBytes(buffer32)
		copy( transactions[t].TxId[:], buffer32 )

		// create wtxid
		pass = sha256.Sum256(wtxidData)
		copy( buffer32, pass[:] )
		pass = sha256.Sum256( buffer32 )
		copy( buffer32, pass[:] )
		ReverseBytes(buffer32)
		copy( transactions[t].WtxId[:], buffer32 )

		transactions[t].TxIdString = fmt.Sprintf("%x", transactions[t].TxId )
		transactions[t].WtxIdString = fmt.Sprintf("%x", transactions[t].WtxId )
	}

	return transactions,bytesUsed,nil
}

func readCount( b byte, file *os.File ) (uint64,int,[]byte,error) {
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
			return 0,0,rawBytes,err
		}
		bytesUsed += skipped

		rawBytes = bytes
		copy(buffer8[0:byteCount], bytes)
		for i:=0; i<8-byteCount; i++  {
			buffer8[i+byteCount]=0
		}

		val = binary.LittleEndian.Uint64(buffer8)
	}
	return val,bytesUsed,rawBytes,nil
}


func ReverseBytes(bytes []byte) {
	for i, j := 0, len(bytes)-1; i < j; i, j = i+1, j-1 {
		bytes[i], bytes[j] = bytes[j], bytes[i]
	}
}

