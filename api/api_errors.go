package api

import "github.com/filecoin-project/go-jsonrpc"

const (
	EOutOfGas = iota + jsonrpc.FirstUserCode
)

type ErrOutOfGas struct{}

func (e ErrOutOfGas) Error() string {
	return "call ran out of gas"
}

var RPCErrors = jsonrpc.NewErrors()

func init() {
	RPCErrors.Register(EOutOfGas, new(ErrOutOfGas))
}
