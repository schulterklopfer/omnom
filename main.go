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
	"cyphernode_indexer/bitcoinBlockchainParser"
	"fmt"
	"path"

	//"github.com/btcsuite/btcd/chaincfg/chainhash"
	//"github.com/btcsuite/btcd/rpcclient"
	//"github.com/jmhodges/levigo"
	_ "github.com/mattn/go-sqlite3"
	//"time"
)

const SQLUpsertAddress = "INSERT INTO address(address,balance) VALUES(?,?) ON CONFLICT(address) DO UPDATE SET balance=balance+excluded.balance;"
const SQLClearAddress = "INSERT INTO address(address,balance) VALUES(?,0) ON CONFLICT(address) DO UPDATE SET balance=0;"
const SQLInsertTx = "INSERT INTO tx(txid,block_hash,hash,timereceived,size,vsize) VALUES(?,?,?,?,?,?);"
const SQLUpdateTxFeeAmount = "UPDATE tx SET fee=?, amount=? WHERE id=?;"
const SQLInsertAddressTx = "INSERT INTO address_tx(address_id,transaction_id) VALUES(?,?);"

func onBlockCallback( block bitcoinBlockchainParser.Block ) {
	fmt.Println( block )
}

func main() {



	bitcoinBlockchainParser := bitcoinBlockchainParser.NewBitcoinBlockchainParser(path.Join("testnet3", "blocks"),onBlockCallback)
	defer bitcoinBlockchainParser.Close()

	bitcoinBlockchainParser.ParseBlocks()

	/*
	db, err := sql.Open("sqlite3", "file:index.sqlite")
	defer db.Close();

	if err != nil {
		panic("ARGL!")
	}
	sqlTx := db//, err := db.Begin()

	if err != nil {
		panic("ARGL!")
	}

	//sqlTx := db//, err := db.Begin()

	if err != nil {
		fmt.Println( err)
	}

	// block hash of tip
	genesisBlockHash,_ := bitcoinBlockchain.GetBlockHash(0)
	genesisBlock,_ := bitcoinBlockchain.GetBlockVerbose(genesisBlockHash)

	currentBlock := genesisBlock
	start := time.Now()

	SQLUpsertAddressStmt, err := sqlTx.Prepare(SQLUpsertAddress)
	if err != nil {
		fmt.Println( err)
	}
	SQLClearAddressStmt, err := sqlTx.Prepare(SQLClearAddress)
	if err != nil {
		fmt.Println( err)
	}
	SQLInsertTxStmt, err := sqlTx.Prepare(SQLInsertTx)
	if err != nil {
		fmt.Println( err)
	}
	SQLUpdateTxFeeAmountStmt, err := sqlTx.Prepare(SQLUpdateTxFeeAmount)
	if err != nil {
		fmt.Println( err)
	}
	SQLInsertAddressTxStmt, err := sqlTx.Prepare(SQLInsertAddressTx)
	if err != nil {
		fmt.Println( err)
	}

	defer SQLUpsertAddressStmt.Close()
	defer SQLClearAddressStmt.Close()
	defer SQLInsertTxStmt.Close()
	defer SQLUpdateTxFeeAmountStmt.Close()
	defer SQLInsertAddressTxStmt.Close()


	blockCount := 0
	for true {
		// anaylse current block

		//fmt.Printf("Block[%.8d]: %s (%d)\n", currentBlock.Height, currentBlock.Hash, len(currentBlock.Tx))
		txCount := len(currentBlock.Tx)

		for i:=0; i<txCount; i++ {
			hash,_ := chainhash.NewHashFromStr(currentBlock.Tx[i])
			tx,_:= bitcoinBlockchain.GetRawTransactionVerbose( hash )
			if tx == nil {
				continue
			}

			r, err := SQLInsertTxStmt.Exec( tx.Txid, tx.BlockHash, tx.Hash, tx.Time, tx.Size, tx.Vsize )
			if err != nil {
				fmt.Println( err)
			}
			sqlTxId,err:=r.LastInsertId()

			txInCount:=len(tx.Vin)
			txOutCount:=len(tx.Vout)
			inSum := int64(0)
			outSum := int64(0)

			for j:=0; j<txInCount; j++ {
				txIn := tx.Vin[j]
				prevTxHash,_ := chainhash.NewHashFromStr(txIn.Txid)
				prevTx,_ := bitcoinBlockchain.GetRawTransaction(prevTxHash)
				if prevTx == nil {
					continue
				}
				//fmt.Println( prevTxHash.String() )
				prevTxOutIndex := txIn.Vout
				msg := prevTx.MsgTx()
				out := msg.TxOut[prevTxOutIndex]
				inSum += out.Value
				script,_ := bitcoinBlockchain.DecodeScript(out.PkScript)
				sourceAddresses := script.Addresses

				// reset balances of output addresses to 0
				// Is that always true? what about address reuse after spending?

				for k:=0; k<len(sourceAddresses); k++ {
					address:=sourceAddresses[k]
					r,err := SQLClearAddressStmt.Exec( address )
					if err != nil {
						fmt.Println( err)
					}
					sqlAddressId,err := r.LastInsertId()
					if err != nil {
						fmt.Println( err)
					}
					_,err = SQLInsertAddressTxStmt.Exec( sqlAddressId, sqlTxId )
					if err != nil {
						fmt.Println( err)
					}
				}
			}

			for j:=0; j<txOutCount; j++ {
				txOut := tx.Vout[j]
				balance := int64(txOut.Value*100000000.0)
				outSum += balance
				targetAddresses := txOut.ScriptPubKey.Addresses
				if targetAddresses != nil {
					r, err := SQLUpsertAddressStmt.Exec( targetAddresses[0], balance )
					if err != nil {
						fmt.Println( err)
					}
					sqlAddressId,err := r.LastInsertId()
					if err != nil {
						fmt.Println( err)
					}
					_,err = SQLInsertAddressTxStmt.Exec( sqlAddressId, sqlTxId )
					if err != nil {
						fmt.Println( err)
					}
				}
				//fmt.Println(err, targetAddresses, txOut)
			}

			_, err = SQLUpdateTxFeeAmountStmt.Exec( inSum-outSum, outSum, sqlTxId )
			if err != nil {
				fmt.Println( err)
			}
		}

		// set next block to visit
		if currentBlock.NextHash != "" {
			hash,_ := chainhash.NewHashFromStr( currentBlock.NextHash )
			currentBlock,_ = bitcoinBlockchain.GetBlockVerbose( hash )
			blockCount++
		} else {
			break
		}

		if blockCount%1000 == 0 {
			elapsed := time.Since(start)
			fmt.Printf("Current height: %.8d\n", currentBlock.Height )
			fmt.Printf("Traversing %d blocks took: %s\n", blockCount, elapsed)
			fmt.Printf("Traversing 1 block took: %s\n", time.Duration(elapsed.Nanoseconds()/int64(blockCount)))
			fmt.Printf("Number of blocks visited: %d\n", blockCount)
		}
		//if blockCount >= 500 {
		//	break
		//}
	}

	//sqlTx.Commit()
	elapsed := time.Since(start)
	fmt.Printf("Traversing all blocks took: %s\n", elapsed)
	fmt.Printf("   Traversing 1 block took: %s\n", time.Duration(elapsed.Nanoseconds()/int64(blockCount)))
	fmt.Printf("  Number of blocks visited: %d\n", blockCount)
	*/
}