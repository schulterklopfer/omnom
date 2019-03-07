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

package addressTxSqlite3Index

import (
	"omnom/bitcoinBlockchainParser"
	"database/sql"
	"fmt"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/txscript"
	_ "github.com/mattn/go-sqlite3"
	"time"
)

type AddressTxSqlite3Index struct {
	//db and statements

	db *sql.DB
	sqlTx *sql.Tx

	sqlInsertTxStmt         *sql.Stmt
	sqlInsertAddressStmt    *sql.Stmt
	sqlSelectAddressStmt    *sql.Stmt
	sqlInsertTxAddressStmt  *sql.Stmt

	//vars
	sqlTxId int64
	chainCfg *chaincfg.Params

	dbName string

}

func NewAddressTxSqlite3Index( chainCfg *chaincfg.Params) *AddressTxSqlite3Index {
	index := new(AddressTxSqlite3Index)
	index.chainCfg = chainCfg
	return index
}

func ( indexer *AddressTxSqlite3Index ) DBName() string {
	return indexer.dbName
}

func ( indexer *AddressTxSqlite3Index) OnStart() (bool,error)  {

	indexer.dbName = fmt.Sprintf("fullIndex-%d.sqlite", time.Now().Unix() )


	var err error
	indexer.db, err = sql.Open("sqlite3", "file:"+indexer.dbName )
	if err != nil {
		return false,err
	}

	// do onStart statements here
	_,err = indexer.db.Exec(SQLOnStart)
	if err != nil {
		return false,err
	}
	indexer.sqlTx,err = indexer.db.Begin()
	if err != nil {
		return false,err
	}
	indexer.sqlInsertTxAddressStmt, err = indexer.sqlTx.Prepare(SQLInsertTxAddress)
	if err != nil {
		return false,err
	}
	indexer.sqlInsertAddressStmt, err = indexer.sqlTx.Prepare(SQLInsertAddress)
	if err != nil {
		return false,err
	}
	indexer.sqlSelectAddressStmt, err = indexer.sqlTx.Prepare(SQLSelectAddress)
	if err != nil {
		return false,err
	}
	indexer.sqlInsertTxStmt, err = indexer.sqlTx.Prepare(SQLInsertTx)
	if err != nil {
		return false,err
	}
	return true,nil
}

func ( indexer *AddressTxSqlite3Index) OnEnd() error {
	err := indexer.sqlTx.Commit()
	if err != nil {
		return err
	}
	err = indexer.sqlInsertTxAddressStmt.Close()
	if err != nil {
		return err
	}
	err = indexer.sqlInsertAddressStmt.Close()
	if err != nil {
		return err
	}
	err = indexer.sqlSelectAddressStmt.Close()
	if err != nil {
		return err
	}
	err = indexer.sqlInsertTxStmt.Close()
	if err != nil {
		return err
	}

	err = indexer.db.Close()
	if err != nil {
		return err
	}

	// do onEnd statements
	_,err = indexer.db.Exec(SQLOnEnd)
	if err != nil {
		return err
	}

	return nil

}

func ( indexer *AddressTxSqlite3Index) OnBlockInfo( height int, total int, blockInfo *bitcoinBlockchainParser.BlockInfo ) error {
	return nil
}

func ( indexer *AddressTxSqlite3Index) OnBlock( height int, total int, currentBlock *bitcoinBlockchainParser.Block ) error {
	// anaylse current block

	// insert block into db
	txCount := len(currentBlock.Transactions)
	for i:=0; i<txCount; i++ {
		r, err := indexer.sqlInsertTxStmt.Exec(
			currentBlock.Transactions[i].TxIdString(),
			currentBlock.Transactions[i].WtxIdString(),
			currentBlock.HashString(),
			currentBlock.Transactions[i].Locktime,
			currentBlock.Transactions[i].Size,
			currentBlock.Transactions[i].VirtualSize,
			currentBlock.Transactions[i].Weight,
			currentBlock.Transactions[i].BaseSize)

		if err != nil {
			return err
		}

		/*
		if currentBlock.Transactions[i].TxIdString()=="a6911033ace9a1ac6f22d472a22df7762a535d3bc751d94aeeab713eebab64d7"  {
			fmt.Println("found tx");
		}
		*/

		indexer.sqlTxId,err=r.LastInsertId()

		txOutCount:=len(currentBlock.Transactions[i].Outputs)

		for j:=0; j<txOutCount; j++ {

			if currentBlock.Transactions[i].Outputs[j].Script == nil ||
				currentBlock.Transactions[i].Outputs[j].Script.Data == nil ||
				len(currentBlock.Transactions[i].Outputs[j].Script.Data) == 0 {
				continue
			}

			/*
			if currentBlock.Transactions[i].TxIdString()=="a6911033ace9a1ac6f22d472a22df7762a535d3bc751d94aeeab713eebab64d7" {
				fmt.Printf("%x\n", currentBlock.Transactions[i].Outputs[j].Script.Data)
				fmt.Println( currentBlock.Transactions[i].Outputs[j].Script.Hex)
			}
			*/
			_,targetAddresses,_,_ := txscript.ExtractPkScriptAddrs(currentBlock.Transactions[i].Outputs[j].Script.Data, indexer.chainCfg )

			if targetAddresses != nil && len(targetAddresses) > 0 {
				for k:=0; k< len(targetAddresses); k++ {
					address := targetAddresses[k].EncodeAddress()

					var addressId int64
					err := indexer.sqlSelectAddressStmt.QueryRow( address ).
						Scan( &addressId )

					if addressId == 0 {
						r, err = indexer.sqlInsertAddressStmt.Exec( address )
						if err != nil {
							return err
						}
						addressId,err = r.LastInsertId()
						if err != nil {
							return err
						}
					}

					_, err = indexer.sqlInsertTxAddressStmt.Exec( indexer.sqlTxId, addressId )
					if err != nil {
						return err
					}
				}
			}
		}
	}
	return nil
}

func ( indexer *AddressTxSqlite3Index) ShouldParseBlockInfo() bool { return true }
func ( indexer *AddressTxSqlite3Index) ShouldParseBlockBody() bool { return true }
