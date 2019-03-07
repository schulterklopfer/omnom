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

package indexer

import (
  "omnom/bitcoinBlockchainParser"
)

type IndexSearch interface {
  FindTransactionIdsByAddress(address string) ([][]byte, error)
  FindAddressesByTransactionId(txid string) ([][]byte, error)
  FindTransactionIdsByBlockHash(blockHash []byte) ([][32]byte, error)
  FindTransactionIdsByBlockHeight(blockHeight int) ([][]byte, error)
  FindBlockHashByBlockHeight(blockHeight int) ([]byte, error)
  FindBlockInfoByBlockHash(blockHash []byte) (*bitcoinBlockchainParser.BlockInfo, error)
}
