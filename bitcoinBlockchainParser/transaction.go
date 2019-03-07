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

import "fmt"

type Transaction struct {
  TxId         [32]byte
  WtxId        [32]byte
  Version      uint32
  Witness      bool
  Size         int
  BaseSize     int
  VirtualSize  int
  Weight       int
  Amount       uint64
  Fee          uint64
  Inputs       []TxInput
  Outputs      []TxOutput
  WitnessItems []WitnessItem
  Locktime     uint32
}

func (tx *Transaction) WtxIdString() string {
  return fmt.Sprintf("%x", tx.WtxId)
}

func (tx *Transaction) TxIdString() string {
  return fmt.Sprintf("%x", tx.TxId)
}

type WitnessItem struct {
  Data []byte

  BlkFilePosition int
}

type TxInput struct {
  SourceTxHash [32]byte
  OutputIndex  uint32
  Script       []byte
  Sequence     uint32
}

func (txi *TxInput) SourceTxHashString() string {
  return fmt.Sprintf("%x", txi.SourceTxHash)
}

type TxOutput struct {
  Value  uint64
  Script *Script
}
