package bitcoinBlockchainParser

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"
	"strings"
	"time"
)

type BitcoinBlockchainParser struct {
	// private
	directory string
	onBlock onBlockCallback
}

type onBlockCallback func( Block )

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

}

type Transaction struct {
	Version uint32
	Witness bool
	Inputs []TxInput
	Outputs []TxOutput
	WitnessItems []WitnessItem
	Locktime uint32
}

type WitnessItem struct {
	Data []byte
}

type TxInput struct {
	SourceTxHash [32]byte
	OutputIndex uint32
	Script []byte
	Sequence uint32
}

type TxOutput struct {
	Value uint64
	Script []byte

}



func NewBitcoinBlockchainParser( directory string,  onBlock onBlockCallback ) *BitcoinBlockchainParser {
	return &BitcoinBlockchainParser{directory, onBlock }
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

var BYTES_READ int

// todo: use standard length buffers for 4,8,32 and only alloc for variable lengths exceeding 4096 bytes
var buffer4 = make([]byte,4)
var buffer8 = make([]byte,8)
var buffer32 = make([]byte,32)
var buffer4096 = make([]byte,32)

func (bc *BitcoinBlockchainParser ) ParseBlocks() {
	fileInfos, err := ioutil.ReadDir(bc.directory)
	if err != nil {
		log.Fatal(err)
		return
	}

	fileInfos = filterBlockDataFiles(fileInfos)
	start := time.Now()
	blockCount := 0

	for index, fileInfo := range fileInfos {
		fmt.Printf("Opening %s [%d of %d]\n", fileInfo.Name(), index+1, len(fileInfos) )

		// Open readonly
		file, err := os.Open( path.Join(bc.directory, fileInfo.Name()))
		if err != nil {
			log.Fatal(err)
			return
		}
		BYTES_READ = 0
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
			blockCount++
			nextBlockPosition += int64(block.Size)+8
			_, err = file.Seek(nextBlockPosition,0)

			if err != nil {
				break
			}

			if blockCount%10000 == 0 {
				elapsed := time.Since(start)
				fmt.Printf("Traversing %d blocks took: %s\n", blockCount, elapsed)
				fmt.Printf("Traversing 1 block took: %s\n", time.Duration(elapsed.Nanoseconds()/int64(blockCount)))
				fmt.Printf("Number of blocks visited: %d\n", blockCount)
				fmt.Printf("Bytes processsed: %d of %d \n", nextBlockPosition, fileInfo.Size() )
				fmt.Printf("%x %d\n", block.Hash, len(block.Transactions) )
				fmt.Println("---")

			}
		}
		fmt.Printf("Done: %d of %d bytes, %d blocks \n", nextBlockPosition, fileInfo.Size(), blockCount )
		fmt.Println("---")

		file.Close()
	}
	fmt.Printf("Traversing all blocks took: %s\n", blockCount, time.Since(start))
}

func (bc *BitcoinBlockchainParser) parseBlock( file *os.File ) (*Block, int, error) {

	block := new(Block)
	bytesUsed := 0
	var bytes []byte
	var err error
	var skipped int

	// Read first 4 bytes of blockdata
	// TODO: What is this?! Maybe some block marker
	bytes = make([]byte,4)
	skipped, err = file.Read(bytes)
	if err != nil || skipped != 4 {
		fmt.Println("Skip")
		return nil,0,err
	}
	bytesUsed += skipped
	BYTES_READ+=skipped

	// Size
	bytes = make([]byte, 4)
	skipped,err = file.Read(bytes)
	if err != nil {
		fmt.Println("Read size")
		return nil,0,err
	}
	block.Size = binary.LittleEndian.Uint32(bytes)
	bytesUsed += skipped
	BYTES_READ+=skipped


	// Header
	/* Read next 80 bytes which will contain
		* version (4 bytes)
        * hash of previous block (32 bytes)
		* merkle root (32 bytes)
        * time stamp (4 bytes)
	    * difficulty (4 bytes)
		* nonce (4 bytes)
	*/
	bytes = make([]byte,80)
	skipped, err = file.Read(bytes)
	if err != nil || skipped != 80 {
		fmt.Println("Read header")
		return nil,0,err
	}
	bytesUsed += skipped
	BYTES_READ+=skipped

	block.Version = binary.LittleEndian.Uint32(bytes[0:4])
	copy(block.PrevHash[:], bytes[4:36])
	copy(block.MerkleRoot[:], bytes[36:68])
	block.Timestamp = binary.LittleEndian.Uint32(bytes[68:72])
	copy(block.Difficulty[:], bytes[72:76])
	block.Nonce = binary.LittleEndian.Uint32(bytes[76:80])

	// Create block hash from those 80 bytes
	blockHash := make( []byte,32 )
	pass := sha256.Sum256(bytes)
	copy( blockHash, pass[:] )
	pass = sha256.Sum256( blockHash )
	copy( blockHash, pass[:] )
	reverseBytes(blockHash)
	copy( block.Hash[:], blockHash )

	// Transaction count bytes
	bytes = make([]byte,1)
	skipped, err = file.Read(bytes)
	if err != nil || skipped != 1 {
		fmt.Println("Read tx count")
		return nil,0,err
	}
	bytesUsed+=skipped
	BYTES_READ+=skipped

	txCount, txCountBytesUsed, _, err := readCount(bytes[0], file)

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
	var bytes []byte
	var err error
	var skipped int

	for t:=0; t<transactionCount; t++ {
		rawTx := make([]byte,0)
		// Version
		bytes = make( []byte, 4)
		skipped, err = file.Read(bytes )
		if err != nil || skipped != 4 {
			fmt.Println("Read version")
			return nil,0,err
		}
		bytesUsed += skipped
		BYTES_READ+=skipped

		transactions[t].Version = binary.LittleEndian.Uint32(bytes)

		reverseBytes(bytes)
		rawTx = append( rawTx, bytes... )

		bytes = make( []byte, 1)
		skipped, err = file.Read(bytes)
		if err != nil || skipped != 1 {
			fmt.Println("Read input count 0")
			return nil,0,err
		}
		bytesUsed += skipped
		BYTES_READ+=skipped

		b := bytes[0]

		// is witness flag present?
		// 0 says yes, cause there are no tx with 0 inputs
		if b==0 {
			bytes = make( []byte, 2)
			skipped, err = file.Read(bytes)
			if err != nil || skipped != 2 {
				fmt.Println("Read input count 1")
				return nil,0,err
			}
			bytesUsed += skipped
			BYTES_READ+=skipped

			b = bytes[1]

			transactions[t].Witness = true
		}

		inputCount,inputCountBytesUsed,rawBytes,err := readCount(b,file)
		bytesUsed+=inputCountBytesUsed

		// TODO: the following might be wrong :)
		reverseBytes(rawBytes)
		rawTx = append( rawTx, rawBytes... )

		if err != nil {
			fmt.Println("input count")
			return nil,0,err
		}

		transactions[t].Inputs = make([]TxInput,inputCount)

		for i:=0; i<int(inputCount); i++ {

			// Source tx hash
			bytes = make( []byte, 32)
			skipped, err = file.Read(bytes )
			if err != nil || skipped != 32 {
				fmt.Println("Read input hash")
				return nil,0,err
			}
			bytesUsed += skipped
			BYTES_READ+=skipped


			copy(transactions[t].Inputs[i].SourceTxHash[:],bytes)
			reverseBytes(bytes)
			rawTx = append( rawTx, bytes... )

			// Source tx output index
			bytes = make( []byte, 4)
			skipped, err = file.Read(bytes)
			if err != nil || skipped != 4 {
				fmt.Println("Read input index")
				return nil,0,err
			}
			bytesUsed += skipped
			BYTES_READ+=skipped


			transactions[t].Inputs[i].OutputIndex = binary.LittleEndian.Uint32(bytes)
			reverseBytes(bytes)
			rawTx = append( rawTx, bytes... )

			// Script length
			bytes = make( []byte, 1)
			skipped, err = file.Read(bytes)
			if err != nil || skipped != 1 {
				fmt.Println("Read script length")
				return nil,0,err
			}
			bytesUsed += skipped
			BYTES_READ+=skipped

			scriptLength,scriptLengthBytesUsed,rawBytes,err := readCount(bytes[0], file)
			bytesUsed += scriptLengthBytesUsed

			// TODO: prolly broken
			reverseBytes(rawBytes)
			rawTx = append( rawTx, rawBytes...)


			// Script
			if scriptLength > 0 {
				bytes = make( []byte, scriptLength)
				skipped, err = file.Read(bytes)
				if err != nil || skipped != int(scriptLength) {
					fmt.Println("Read input script", err)
					return nil,0,err
				}
				bytesUsed += skipped
				BYTES_READ+=skipped

				transactions[t].Inputs[i].Script = bytes

				// TODO: prolly broken
				reverseBytes(bytes)
				rawTx = append( rawTx, bytes...)
			}

			// Sequence
			bytes = make( []byte, 4)
			skipped, err = file.Read(bytes)
			if err != nil || skipped != 4 {
				fmt.Println("Read sequence")
				return nil,0,err
			}
			bytesUsed += skipped
			BYTES_READ+=skipped


			transactions[t].Inputs[i].Sequence = binary.LittleEndian.Uint32(bytes)

			// TODO: prolly broken
			reverseBytes(bytes)
			rawTx = append( rawTx, bytes... )

		}

		// Output count
		bytes = make( []byte, 1)
		skipped, err = file.Read(bytes)
		if err != nil || skipped != 1 {
			fmt.Println("Peek output count")
			return nil,0,err
		}
		bytesUsed += skipped
		BYTES_READ+=skipped


		outputCount,outputCountBytesUsed,rawBytes,err := readCount(bytes[0], file)
		if err != nil {
			fmt.Println("output count")
			return nil,0,err
		}

		bytesUsed += outputCountBytesUsed

		// TODO: the following might be wrong :)
		reverseBytes(rawBytes)
		rawTx = append( rawTx, rawBytes... )


		transactions[t].Outputs = make([]TxOutput,outputCount)

		for o:=0; o<int(outputCount); o++ {

			// Value
			bytes = make( []byte, 8)
			skipped, err = file.Read(bytes)
			if err != nil || skipped != 8 {
				fmt.Println("Read value")
				return nil,0,err
			}
			bytesUsed += skipped
			BYTES_READ+=skipped


			transactions[t].Outputs[o].Value = binary.LittleEndian.Uint64(bytes)

			// TODO: the following might be wrong :)
			reverseBytes(bytes)
			rawTx = append( rawTx, bytes... )

			// Script length
			bytes = make( []byte, 1)
			skipped, err = file.Read(bytes)
			if err != nil || skipped != 1 {
				fmt.Println("Peek script length")
				return nil,0,err
			}
			bytesUsed += skipped
			BYTES_READ+=skipped


			scriptLength,scriptLengthBytesUsed,rawBytes,err := readCount(bytes[0], file)
			if err != nil {
				fmt.Println("script length")
				return nil,0,err
			}

			bytesUsed += scriptLengthBytesUsed

			// TODO: the following might be wrong :)
			reverseBytes(rawBytes)
			rawTx = append( rawTx, rawBytes... )

			// Script
			if scriptLength > 0 {
				bytes = make( []byte, scriptLength)
				skipped, err = file.Read(bytes)
				if err != nil || skipped != int(scriptLength) {
					fmt.Println("Read output script", err)
					return nil,0,err
				}
				bytesUsed += skipped
				BYTES_READ+=skipped

				transactions[t].Outputs[o].Script = bytes

				// TODO: prolly broken
				reverseBytes(bytes)
				rawTx = append( rawTx, bytes...)
			}
		}

		if transactions[t].Witness {
			// Witness length
			for i:=0; i<int(inputCount); i++ {
				bytes = make( []byte, 1)
				skipped, err = file.Read(bytes)
				if err != nil || skipped != 1 {
					fmt.Println("Read witness length")
					return nil,0,err
				}
				bytesUsed += skipped
				BYTES_READ+=skipped

				witnessLength,witnessLengthBytesUsed,_,err := readCount(bytes[0], file)
				if err != nil {
					fmt.Println("witness length")
					return nil,0,err
				}

				bytesUsed += witnessLengthBytesUsed

				// Witness
				transactions[t].WitnessItems = make([]WitnessItem,witnessLength)
				for w:=0; w<int(witnessLength); w++ {
					// Witness item length
					bytes = make( []byte, 1)
					skipped, err = file.Read(bytes)
					if err != nil || skipped != 1 {
						fmt.Println("Read witness item length")
						return nil,0,err
					}
					bytesUsed += skipped
					BYTES_READ+=skipped

					witnessItemLength,witnessItemLengthBytesUsed,_,err := readCount(bytes[0], file)
					if err != nil {
						fmt.Println("witness item length")
						return nil,0,err
					}

					bytesUsed += witnessItemLengthBytesUsed
					bytes = make([]byte,witnessItemLength)
					//skipped64, err := file.Seek(int64(witnessItemLength),1)
					skipped, err = file.Read(bytes)
					if err != nil || skipped != int(witnessItemLength) {
						fmt.Println("Read witness")
						return nil,0,err
					}
					transactions[t].WitnessItems[w].Data = bytes
					bytesUsed += skipped
					BYTES_READ+=skipped

				}
			}
		}

		// Lock time
		bytes = make([]byte,4)
		skipped, err = file.Read(bytes)
		if err != nil || skipped != 4 {
			fmt.Println("Read lock time")
			return nil,0,err
		}
		bytesUsed += skipped
		BYTES_READ+=skipped


		transactions[t].Locktime = binary.LittleEndian.Uint32(bytes)

		// TODO: the following might be wrong :)
		reverseBytes(bytes)
		rawTx = append( rawTx, bytes... )
	}

	return transactions,bytesUsed,nil
}

func readCount( b byte, file *os.File ) (uint64,int,[]byte,error) {
	bytesUsed := int(0)

	val := uint64(0)
	var rawBytes []byte

	if b < 253 {
		val = uint64(b)
		rawBytes = []byte{b}
	} else {
		byteCount := 0

		uint64bytes := make( []byte,8 )
		if b == 253 {
			byteCount = 2
		} else if b == 254 {
			byteCount = 4
		} else if b == 255 {
			byteCount = 8
		}
		bytes := make( []byte,byteCount )

		skipped, err := file.Read(bytes)
		if err != nil || skipped != byteCount {
			fmt.Println("Read count 1")
			return 0,0,rawBytes,err
		}
		bytesUsed += skipped
		BYTES_READ+=skipped


		// TODO: check if I have to revers rawBytes
		rawBytes = bytes
		copy(uint64bytes[0:byteCount], bytes)
		val = binary.LittleEndian.Uint64(uint64bytes)
	}
	return val,bytesUsed,rawBytes,nil
}

func reverseBytes(bytes []byte) {
	for i, j := 0, len(bytes)-1; i < j; i, j = i+1, j-1 {
		bytes[i], bytes[j] = bytes[j], bytes[i]
	}
}

func obfuscateBytes(bytes []byte, obfuscateKey []byte ) {
	byteCount := len( bytes )
	keySize := len( obfuscateKey )

	if keySize == 0 {
		return
	}

	for i, j := 0, 0; i < byteCount; i++ {
		// XOR with reepeating obfuscateKey
		bytes[i] ^= obfuscateKey[j]
		j++
		if j == keySize {
			j = 0
		}
	}
}
