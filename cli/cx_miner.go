package cli

import (
	"encoding/json"
	"fmt"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/big"
	"github.com/filecoin-project/lotus/api"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/urfave/cli/v2"
)

// MinerState
type MinerFullData struct {
	Address      address.Address
	MinerBalance *MinerBalance
	MinerPower   *api.MinerPower
	MinerSectors api.MinerSectors
	MinerInfo    api.MinerInfo
}

type MinerBalance struct {
	Balance          abi.TokenAmount
	AvailableBalance abi.TokenAmount
	InitialPledge    abi.TokenAmount
	LockedRewards    abi.TokenAmount
}

var MinerExCmd = &cli.Command{
	Name:  "miner",
	Usage: "Miner with filecoin blockchain",
	Subcommands: []*cli.Command{
		MinerListCmd,
		MinerStateCmd,
		MinerSectorCmd,
	},
}

// MinerListCmd  矿工列表
var MinerListCmd = &cli.Command{
	Name:      "list",
	Usage:     "Miner list",
	ArgsUsage: "[miner address]",
	Action: func(cctx *cli.Context) error {
		api, closer, err := GetFullNodeAPI(cctx)
		if err != nil {
			return err
		}
		defer closer()

		ctx := ReqContext(cctx)

		ts, err := LoadTipSet(ctx, cctx, api)
		if err != nil {
			return err
		}

		miners, err := api.StateListMiners(ctx, ts.Key())
		if err != nil {
			return err
		}

		out, err := json.MarshalIndent(miners, "", "  ")
		if err != nil {
			return err
		}

		afmt := NewAppFmt(cctx.App)
		afmt.Println(string(out))

		return nil
	},
}

var MinerStateCmd = &cli.Command{
	Name:      "state",
	Usage:     "Miner state",
	ArgsUsage: "[miner address]",

	Action: func(cctx *cli.Context) error {
		api, closer, err := GetFullNodeAPI(cctx)
		if err != nil {
			return err
		}
		defer closer()

		ctx := ReqContext(cctx)

		if !cctx.Args().Present() {
			return fmt.Errorf("must specify miner to show power for")
		}

		ts, err := LoadTipSet(ctx, cctx, api)
		if err != nil {
			return err
		}

		var minerFullData MinerFullData

		maddr, err := address.NewFromString(cctx.Args().First())
		if err != nil {
			return err
		}

		newfaddr, err := api.StateAccountKey(ctx, maddr, types.EmptyTSK)
		if err == nil {
			maddr = newfaddr
		}
		minerFullData.Address = maddr

		walletBalance, err := api.WalletBalance(ctx, maddr)
		if err != nil {
			return err
		}
		minerFullData.MinerBalance.Balance = walletBalance

		availableBalance, err := api.StateMinerAvailableBalance(ctx, maddr, ts.Key())
		if err != nil {
			return err
		}
		minerFullData.MinerBalance.AvailableBalance = availableBalance

		sectors, err := api.StateMinerSectors(ctx, maddr, nil, ts.Key())
		if err != nil {
			return err
		}

		minerFullData.MinerBalance.InitialPledge = big.Zero()

		for _, s := range sectors {
			minerFullData.MinerBalance.InitialPledge = big.Add(minerFullData.MinerBalance.InitialPledge, s.InitialPledge)
		}

		minerFullData.MinerBalance.LockedRewards = big.Sub(minerFullData.MinerBalance.Balance,
			big.Add(minerFullData.MinerBalance.AvailableBalance,
				minerFullData.MinerBalance.InitialPledge))

		power, err := api.StateMinerPower(ctx, maddr, ts.Key())
		if err != nil {
			return err
		}
		minerFullData.MinerPower = power

		minerSectors, err := api.StateMinerSectorCount(ctx, maddr, ts.Key())
		if err != nil {
			return err
		}
		minerFullData.MinerSectors = minerSectors

		minerInfo, err := api.StateMinerInfo(ctx, maddr, ts.Key())
		if err != nil {
			return err
		}
		minerFullData.MinerInfo = minerInfo

		out, err := json.MarshalIndent(minerFullData, "", "  ")
		if err != nil {
			return err
		}

		afmt := NewAppFmt(cctx.App)
		afmt.Println(string(out))

		return nil
	},
}
