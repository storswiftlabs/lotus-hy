package cli

import (
	"encoding/json"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/lotus/chain/actors/builtin"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/filecoin-project/lotus/chain/types/ethtypes"
	"github.com/urfave/cli/v2"
	"golang.org/x/xerrors"
)

type EXAddressDescription struct {
	ID       address.Address
	Filecoin address.Address
	Eth      ethtypes.EthAddress
	Type     string
}

var ExCmd = &cli.Command{
	Name:  "ex",
	Usage: "The extension interface to the filecoin browser project.",
	Subcommands: []*cli.Command{
		ExAddressTransformationCmd,
	},
}

var ExAddressTransformationCmd = &cli.Command{
	Name:      "addr-description",
	Aliases:   []string{"addrdescription"},
	Usage:     "address description",
	ArgsUsage: "address",
	Action: func(cctx *cli.Context) error {
		if argc := cctx.Args().Len(); argc < 1 {
			return xerrors.Errorf("must pass the address")
		}

		api, closer, err := GetFullNodeAPI(cctx)
		if err != nil {
			return err
		}
		defer closer()
		ctx := ReqContext(cctx)

		addrString := cctx.Args().Get(0)

		var out EXAddressDescription

		var faddr address.Address
		var eaddr ethtypes.EthAddress
		addr, err := address.NewFromString(addrString)
		if err != nil { // This isn't a filecoin address
			eaddr, err = ethtypes.ParseEthAddress(addrString)
			if err != nil { // This isn't an Eth address either
				return xerrors.Errorf("address is not a filecoin or eth address")
			}
			faddr, err = eaddr.ToFilecoinAddress()
			if err != nil {
				return err
			}
		} else {
			eaddr, faddr, err = ethAddrFromFilecoinAddress(ctx, addr, api)
			if err != nil {
				return err
			}
		}

		out.Filecoin = faddr
		out.Eth = eaddr

		actor, err := api.StateGetActor(ctx, faddr, types.EmptyTSK)
		if err == nil {
			out.ID = *actor.Address
			out.Type = builtin.ActorNameByCode(actor.Code)
		}

		byte, err := json.MarshalIndent(out, "", "  ")
		if err != nil {
			return err
		}
		afmt := NewAppFmt(cctx.App)
		afmt.Println(string(byte))
		return nil
	},
}
