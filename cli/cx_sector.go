package cli

import (
	"encoding/json"
	"fmt"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/big"
	"github.com/filecoin-project/lotus/chain/actors/builtin/miner"
	"github.com/urfave/cli/v2"
)

type EXMinerSectorsInfo struct {
	MinerAddress             address.Address
	Height                   abi.ChainEpoch
	AllInitialPledge         abi.TokenAmount // All sectors pledge collected to commit this sector
	AllExpectedDayReward     abi.TokenAmount // All sectors expected one day projection of reward for sector computed at activation time
	AllExpectedStoragePledge abi.TokenAmount // All sectors expected twenty day projection of reward for sector computed at activation time
	AllReplacedDayReward     abi.TokenAmount // All sectors day reward of sector this sector replace or zero
	Sectors                  []*miner.SectorOnChainInfo
}

var MinerSectorCmd = &cli.Command{
	Name:      "sectors",
	Aliases:   []string{"sectors"},
	Usage:     "Get miner all sector info",
	ArgsUsage: "[miner address]",
	Flags: []cli.Flag{
		&cli.BoolFlag{
			Name:  "pledge",
			Usage: "print just the miner all sectors pledge collected to commit this sector",
		},
	},
	Action: func(cctx *cli.Context) error {
		api, closer, err := GetFullNodeAPI(cctx)
		if err != nil {
			return err
		}
		defer closer()

		ctx := ReqContext(cctx)

		if !cctx.Args().Present() {
			return fmt.Errorf("must specify miner to list sectors for")
		}

		maddr, err := address.NewFromString(cctx.Args().First())
		if err != nil {
			return err
		}

		ts, err := LoadTipSet(ctx, cctx, api)
		if err != nil {
			return err
		}

		sectors, err := api.StateMinerSectors(ctx, maddr, nil, ts.Key())
		if err != nil {
			return err
		}

		exMinerSectorsInfo := EXMinerSectorsInfo{
			MinerAddress:             maddr,
			Height:                   ts.Height(),
			AllInitialPledge:         big.NewInt(0),
			AllExpectedDayReward:     big.NewInt(0),
			AllExpectedStoragePledge: big.NewInt(0),
			AllReplacedDayReward:     big.NewInt(0),
		}

		
		for _, s := range sectors {
			exMinerSectorsInfo.AllInitialPledge = big.Add(exMinerSectorsInfo.AllInitialPledge, s.InitialPledge)
			exMinerSectorsInfo.AllExpectedDayReward = big.Add(exMinerSectorsInfo.AllExpectedDayReward, s.ExpectedDayReward)
			exMinerSectorsInfo.AllExpectedStoragePledge = big.Add(exMinerSectorsInfo.AllExpectedStoragePledge, s.ExpectedStoragePledge)
			exMinerSectorsInfo.AllReplacedDayReward = big.Add(exMinerSectorsInfo.AllReplacedDayReward, s.ReplacedDayReward)
		}
		
		if !cctx.Bool("pledge") {
			exMinerSectorsInfo.Sectors = sectors
		}

		byte, err := json.MarshalIndent(exMinerSectorsInfo, "", "  ")
		if err != nil {
			return err
		}
		afmt := NewAppFmt(cctx.App)
		afmt.Println(string(byte))

		return nil
	},
}
