package bitcoinBlockchainParser

import (
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcutil"
)

type Script struct {
	Data []byte
	Addresses []btcutil.Address
	Class txscript.ScriptClass
	Required int
}

func NewScript( data []byte ) *Script {
	s := new(Script)
	s.Data = data

	class, addresses, required, err := txscript.ExtractPkScriptAddrs(s.Data,CHAINCFG)
	if err == nil {
		s.Addresses = addresses
		s.Required = required
		s.Class = class
	}
	return s
}
