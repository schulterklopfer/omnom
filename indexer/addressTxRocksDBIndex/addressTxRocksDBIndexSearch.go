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
  "github.com/pkg/errors"
  "github.com/tecbot/gorocksdb"
  "omnom/bitcoinBlockchainParser"
)

type AddressTxRocksDBIndexSearch struct {
  readOptions *gorocksdb.ReadOptions
  cfHandles   []*gorocksdb.ColumnFamilyHandle
  db          *gorocksdb.DB
}

func NewIndexSearch(db *gorocksdb.DB, readOptions *gorocksdb.ReadOptions, cfHandles []*gorocksdb.ColumnFamilyHandle) *AddressTxRocksDBIndexSearch {
  s := new(AddressTxRocksDBIndexSearch)
  s.db = db
  s.cfHandles = cfHandles
  s.readOptions = readOptions
  return s
}

func (s *AddressTxRocksDBIndexSearch) FindTransactionIdsByAddress(address string) ([][]byte, error) {
  return nil, nil
}

func (s *AddressTxRocksDBIndexSearch) FindAddressesByTransactionId(txid string) ([][]byte, error) {
  return nil, nil
}

func (s *AddressTxRocksDBIndexSearch) FindTransactionIdsByBlockHash(blockHash []byte) ([][32]byte, error) {
  bytes, err := s.getBytesCF(blockHash, s.cfHandles[3])

  if err != nil {
    return nil, err
  }

  if len(bytes)%32 != 0 {
    return nil, errors.New("Unexpected result size")
  }

  txCount := int(len(bytes) / 32)

  result := make([][32]byte, txCount)

  for i := 0; i < txCount; i++ {
    copy(result[i][0:32], bytes[i*32:i*32+32])
  }

  return result, nil
}

func (s *AddressTxRocksDBIndexSearch) FindTransactionIdsByBlockHeight(blockHeight int) ([][]byte, error) {
  return nil, nil
}

func (s *AddressTxRocksDBIndexSearch) FindBlockHashByBlockHeight(blockHeight int) ([]byte, error) {
  return nil, nil
}

func (s *AddressTxRocksDBIndexSearch) FindBlockInfoByBlockHash(blockHash []byte) (*bitcoinBlockchainParser.BlockInfo, error) {
  bytes, err := s.getBytesCF(blockHash, s.cfHandles[4])

  if err != nil {
    return nil, err
  }

  blockInfo := bitcoinBlockchainParser.BlockInfoFromBytes(blockHash, bytes, nil)

  return blockInfo, nil
}

func (s *AddressTxRocksDBIndexSearch) getBytesCF(key []byte, cf *gorocksdb.ColumnFamilyHandle) ([]byte, error) {
  txs, err := s.db.GetCF(s.readOptions, cf, key)
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

  result := make([]byte, txs.Size())
  copy(result, txs.Data())

  txs.Free()

  return result, nil

}
