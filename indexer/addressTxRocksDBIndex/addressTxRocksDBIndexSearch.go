package addressTxRocksDBIndex

import (
	"github.com/pkg/errors"
	"github.com/tecbot/gorocksdb"
	"omnom/bitcoinBlockchainParser"
)

type AddressTxRocksDBIndexSearch struct {
	readOptions *gorocksdb.ReadOptions
	cfHandles []*gorocksdb.ColumnFamilyHandle
	db *gorocksdb.DB
}

func NewIndexSearch( db *gorocksdb.DB, readOptions *gorocksdb.ReadOptions, cfHandles []*gorocksdb.ColumnFamilyHandle ) *AddressTxRocksDBIndexSearch {
	s := new(AddressTxRocksDBIndexSearch)
	s.db = db
	s.cfHandles = cfHandles
	s.readOptions = readOptions
	return s
}

func ( s *AddressTxRocksDBIndexSearch) FindTransactionIdsByAddress( address string ) ([][]byte,error)  {
	return nil,nil
}

func ( s *AddressTxRocksDBIndexSearch) FindAddressesByTransactionId( txid string ) ([][]byte,error)  {
	return nil,nil
}

func ( s *AddressTxRocksDBIndexSearch) FindTransactionIdsByBlockHash( blockHash []byte ) ([][32]byte,error) {
	bytes, err := s.getBytesCF( blockHash, s.cfHandles[3] )

	if err != nil {
		return nil, err
	}

	if len(bytes)%32 != 0 {
		return nil, errors.New("Unexpected result size")
	}

	txCount := int(len(bytes)/32)

	result := make([][32]byte, txCount )

	for i:=0; i<txCount; i++ {
		copy( result[i][0:32], bytes[i*32:i*32+32] )
	}

	return result,nil
}

func ( s *AddressTxRocksDBIndexSearch) FindTransactionIdsByBlockHeight( blockHeight int ) ([][]byte,error) {
	return nil,nil
}

func ( s *AddressTxRocksDBIndexSearch) FindBlockHashByBlockHeight( blockHeight int ) ([]byte,error) {
	return nil,nil
}

func ( s *AddressTxRocksDBIndexSearch) FindBlockInfoByBlockHash( blockHash []byte  ) (*bitcoinBlockchainParser.BlockInfo, error) {
	bytes, err := s.getBytesCF( blockHash, s.cfHandles[4] )

	if err != nil {
		return nil, err
	}

	blockInfo := bitcoinBlockchainParser.BlockInfoFromBytes( blockHash, bytes, nil )

	return blockInfo,nil
}

func ( s *AddressTxRocksDBIndexSearch) getBytesCF( key []byte, cf *gorocksdb.ColumnFamilyHandle ) ([]byte, error) {
	txs, err := s.db.GetCF( s.readOptions, cf, key )
	if err != nil {
		if txs != nil {
			txs.Free()
		}
		return nil, err
	}
	if txs.Size() == 0 {
		txs.Free()
		return nil, nil
	}

	result := make( []byte, txs.Size() )
	copy( result, txs.Data() )

	txs.Free()

	return result,nil

}