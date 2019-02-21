package fullSqlite3Index

import (
	"omnom/bitcoinBlockchainParser"
	"database/sql"
	"fmt"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/txscript"
	_ "github.com/mattn/go-sqlite3"
	"time"
)

type FullSqlite3Index struct {
	//db and statements

	db *sql.DB
	sqlTx *sql.Tx

	sqlInsertBlockStmt *sql.Stmt
	sqlInsertInputStmt *sql.Stmt
	sqlInsertOutputStmt *sql.Stmt
	sqlUpdateAddressBalanceStmt *sql.Stmt
	sqlInsertTxStmt *sql.Stmt
	sqlUpdateTxFeeAmountStmt *sql.Stmt
	sqlSelectOutputStmt  *sql.Stmt
	sqlUpsertAddress *sql.Stmt

	//vars
	sqlBlockId int64
	sqlTxId int64
	sqlAddressId int64
	chainCfg *chaincfg.Params
}

func NewFullSqlite3Index( chainCfg *chaincfg.Params) *FullSqlite3Index {
	index := new(FullSqlite3Index)
	index.chainCfg = chainCfg
	return index
}

func ( indexer *FullSqlite3Index) OnStart() error {

	fileName := fmt.Sprintf("fullIndex-%d.sqlite", time.Now().Unix() )


	var err error
	indexer.db, err = sql.Open("sqlite3", "file:"+fileName )
	if err != nil {
		return err
	}

	// do onStart statements here
	_,err = indexer.db.Exec(SQLOnStart)
	if err != nil {
		return err
	}
	indexer.sqlTx,err = indexer.db.Begin()
	if err != nil {
		return err
	}
	indexer.sqlInsertBlockStmt, err = indexer.sqlTx.Prepare(SQLInsertBlock)
	if err != nil {
		return err
	}
	indexer.sqlUpdateAddressBalanceStmt, err = indexer.sqlTx.Prepare(SQLUpdateAddressBalance)
	if err != nil {
		return err
	}
	indexer.sqlUpsertAddress, err = indexer.sqlTx.Prepare(SQLUpsertAddress)
	if err != nil {
		return err
	}
	indexer.sqlInsertTxStmt, err = indexer.sqlTx.Prepare(SQLInsertTx)
	if err != nil {
		return err
	}
	indexer.sqlUpdateTxFeeAmountStmt, err = indexer.sqlTx.Prepare(SQLUpdateTxFeeAmount)
	if err != nil {
		return err
	}
	indexer.sqlInsertInputStmt, err = indexer.sqlTx.Prepare(SQLInsertInput)
	if err != nil {
		return err
	}
	indexer.sqlInsertOutputStmt, err = indexer.sqlTx.Prepare(SQLInsertOutput)
	if err != nil {
		return err
	}
	indexer.sqlSelectOutputStmt, err = indexer.sqlTx.Prepare(SQLSelectOutput)
	if err != nil {
		return err
	}
	return nil
}

func ( indexer *FullSqlite3Index) OnEnd() error {
	err := indexer.sqlTx.Commit()
	if err != nil {
		return err
	}
	err = indexer.sqlInsertBlockStmt.Close()
	if err != nil {
		return err
	}
	err = indexer.sqlInsertInputStmt.Close()
	if err != nil {
		return err
	}
	err = indexer.sqlInsertOutputStmt.Close()
	if err != nil {
		return err
	}
	err = indexer.sqlUpdateAddressBalanceStmt.Close()
	if err != nil {
		return err
	}
	err = indexer.sqlInsertTxStmt.Close()
	if err != nil {
		return err
	}
	err = indexer.sqlUpdateTxFeeAmountStmt.Close()
	if err != nil {
		return err
	}
	err = indexer.sqlSelectOutputStmt.Close()
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

func ( indexer *FullSqlite3Index) OnBlock( height int, total int, currentBlock *bitcoinBlockchainParser.Block ) error {
	// anaylse current block

	// insert block into db
	nextBlockId := int64(0)

	if height < total-1 {
		// infering the nextblock id by adding 2 is safe here, cause the blocks will be added in chronological order
		nextBlockId = indexer.sqlBlockId+2
	}

	r, err := indexer.sqlInsertBlockStmt.Exec( currentBlock.HashString, indexer.sqlBlockId, nextBlockId, currentBlock.Version, currentBlock.Timestamp )
	if err != nil {
		return err
	}
	indexer.sqlBlockId,err=r.LastInsertId()
	if err != nil {
		return err
	}
	txCount := len(currentBlock.Transactions)
	for i:=0; i<txCount; i++ {
		r, err := indexer.sqlInsertTxStmt.Exec(
			currentBlock.Transactions[i].TxIdString,
			indexer.sqlBlockId,
			currentBlock.Transactions[i].WtxIdString,
			currentBlock.Transactions[i].Locktime,
			currentBlock.Transactions[i].Size,
			currentBlock.Transactions[i].VirtualSize,
			currentBlock.Transactions[i].Weight,
			currentBlock.Transactions[i].BaseSize)

		if err != nil {
			return err
		}
		indexer.sqlTxId,err=r.LastInsertId()

		txInCount:=len(currentBlock.Transactions[i].Inputs)
		txOutCount:=len(currentBlock.Transactions[i].Outputs)
		inSum := uint64(0)
		outSum := uint64(0)

		for j:=0; j<txInCount; j++ {
			txIn := currentBlock.Transactions[i].Inputs[j]
			var outputId int
			var outputAmount int64
			var addressId int
			err := indexer.sqlSelectOutputStmt.QueryRow( txIn.SourceTxHashString, txIn.OutputIndex ).
				Scan(&outputId, &outputAmount, &addressId )
			if err == nil {
				return err
			}

			inSum += uint64(outputAmount)

			// Update address. Was created in outputs already
			_,err = indexer.sqlUpdateAddressBalanceStmt.Exec( -outputAmount, addressId )
			if err != nil {
				return err
			}

			_,err = indexer.sqlInsertInputStmt.Exec( indexer.sqlTxId, outputId )
			if err != nil {
				return err
			}
		}

		for j:=0; j<txOutCount; j++ {

			value := currentBlock.Transactions[i].Outputs[j].Value
			outSum += value


			if currentBlock.Transactions[i].Outputs[j].Script == nil ||
				currentBlock.Transactions[i].Outputs[j].Script.Data == nil ||
				len(currentBlock.Transactions[i].Outputs[j].Script.Data) == 0 {
				continue
			}

			_,targetAddresses,_,_ := txscript.ExtractPkScriptAddrs(currentBlock.Transactions[i].Outputs[j].Script.Data, indexer.chainCfg )

			if targetAddresses != nil && len(targetAddresses) > 0 {
				address := targetAddresses[0].EncodeAddress()

				r, err := indexer.sqlUpsertAddress.Exec( address, int(value) )
				if err != nil {
					return err
				}
				indexer.sqlAddressId,err = r.LastInsertId()

				if err != nil {
					return err
				}
				_,err = indexer.sqlInsertOutputStmt.Exec( indexer.sqlTxId, j, value, indexer.sqlAddressId )
				if err != nil {
					return err
				}
			}
			//fmt.Println(err, targetAddresses, txOut)
		}

		_, err = indexer.sqlUpdateTxFeeAmountStmt.Exec( int(inSum-outSum), int(outSum), indexer.sqlTxId )
		if err != nil {
			return err
		}
	}
	return nil
}
