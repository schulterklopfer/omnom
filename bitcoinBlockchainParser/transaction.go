package bitcoinBlockchainParser

import "fmt"

type Transaction struct {
	TxId [32]byte
	WtxId [32]byte
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
	BlkFileIndex int
	BlkFilePosition int64

}

func (tx *Transaction) WtxIdString() string {
	return fmt.Sprintf("%x", tx.WtxId )
}

func (tx *Transaction) TxIdString() string {
	return fmt.Sprintf("%x", tx.TxId )
}

type WitnessItem struct {
	Data []byte
}

type TxInput struct {
	SourceTxHash [32]byte
	OutputIndex uint32
	Script []byte
	Sequence uint32

	BlkFileIndex int
	BlkFilePosition int64

}

func (txi *TxInput) SourceTxHashString() string {
	return fmt.Sprintf("%x", txi.SourceTxHash )
}

type TxOutput struct {
	Value uint64
	Script *Script

	BlkFileIndex int
	BlkFilePosition int64
}
