package cli

import "github.com/urfave/cli/v2"

var MinerExCmd = &cli.Command{
	Name:  "miner",
	Usage: "Miner with filecoin blockchain",
	Subcommands: []*cli.Command{
		MinerStateCmd,
	},
}



var MinerStateCmd = &cli.Command{
	Name:      "state",
	Usage:     "Miner state",
	ArgsUsage: "[miner address]",
	
	Action: func(cctx *cli.Context) error {

	}
}