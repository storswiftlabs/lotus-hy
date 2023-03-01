package cli

import (
	"encoding/json"
	"fmt"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/big"
	"github.com/filecoin-project/go-state-types/builtin"
	"github.com/filecoin-project/go-state-types/builtin/v9/util/smoothing"
	"github.com/filecoin-project/lotus/chain/actors/builtin/miner"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/filecoin-project/specs-actors/v2/actors/util/math"
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
		&cli.Uint64Flag{
			Name:  "epoch",
			Usage: "reset head to given epoch",
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

		if cctx.IsSet("epoch") {
			ts, err = api.ChainGetTipSetByHeight(ctx, abi.ChainEpoch(cctx.Uint64("epoch")), types.EmptyTSK)
		}

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

func terminationPenalty(currEpoch abi.ChainEpoch,
	rewardEstimate, networkQAPowerEstimate smoothing.FilterEstimate, sectors []*miner.SectorOnChainInfo) abi.TokenAmount {
	totalFee := big.Zero()
	for _, s := range sectors {
		sectorSize, _ := s.SealProof.SectorSize()
		sectorPower := QAPowerForSector(sectorSize, s)
		fee := PledgePenaltyForTermination(s.ExpectedDayReward, currEpoch-s.Activation, s.ExpectedStoragePledge,
			networkQAPowerEstimate, sectorPower, rewardEstimate, s.ReplacedDayReward, s.ReplacedSectorAge)
		totalFee = big.Add(fee, totalFee)
	}
	return totalFee
}

// QAPowerForSector The quality-adjusted power for a sector.
func QAPowerForSector(size abi.SectorSize, sector *miner.SectorOnChainInfo) abi.StoragePower {
	duration := sector.Expiration - sector.Activation
	return QAPowerForWeight(size, duration, sector.DealWeight, sector.VerifiedDealWeight)
}

// QAPowerForWeight The power for a sector size, committed duration, and weight.
func QAPowerForWeight(size abi.SectorSize, duration abi.ChainEpoch, dealWeight, verifiedWeight abi.DealWeight) abi.StoragePower {
	quality := QualityForWeight(size, duration, dealWeight, verifiedWeight)
	return big.Rsh(big.Mul(big.NewIntUnsigned(uint64(size)), quality), builtin.SectorQualityPrecision)
}

func QualityForWeight(size abi.SectorSize, duration abi.ChainEpoch, dealWeight, verifiedWeight abi.DealWeight) abi.SectorQuality {
	// sectorSpaceTime = size * duration
	sectorSpaceTime := big.Mul(big.NewIntUnsigned(uint64(size)), big.NewInt(int64(duration)))
	// totalDealSpaceTime = dealWeight + verifiedWeight
	totalDealSpaceTime := big.Add(dealWeight, verifiedWeight)

	// Base - all size * duration of non-deals
	// weightedBaseSpaceTime = (sectorSpaceTime - totalDealSpaceTime) * QualityBaseMultiplier
	weightedBaseSpaceTime := big.Mul(big.Sub(sectorSpaceTime, totalDealSpaceTime), builtin.QualityBaseMultiplier)
	// Deal - all deal size * deal duration * 10
	// weightedDealSpaceTime = dealWeight * DealWeightMultiplier
	weightedDealSpaceTime := big.Mul(dealWeight, builtin.DealWeightMultiplier)
	// Verified - all verified deal size * verified deal duration * 100
	// weightedVerifiedSpaceTime = verifiedWeight * VerifiedDealWeightMultiplier
	weightedVerifiedSpaceTime := big.Mul(verifiedWeight, builtin.VerifiedDealWeightMultiplier)
	// Sum - sum of all spacetime
	// weightedSumSpaceTime = weightedBaseSpaceTime + weightedDealSpaceTime + weightedVerifiedSpaceTime
	weightedSumSpaceTime := big.Sum(weightedBaseSpaceTime, weightedDealSpaceTime, weightedVerifiedSpaceTime)
	// scaledUpWeightedSumSpaceTime = weightedSumSpaceTime * 2^20
	scaledUpWeightedSumSpaceTime := big.Lsh(weightedSumSpaceTime, builtin.SectorQualityPrecision)

	// Average of weighted space time: (scaledUpWeightedSumSpaceTime / sectorSpaceTime * 10)
	return big.Div(big.Div(scaledUpWeightedSumSpaceTime, sectorSpaceTime), builtin.QualityBaseMultiplier)
}

const TerminationLifetimeCap = 140 // PARAM_SPEC
func minEpoch(a, b abi.ChainEpoch) abi.ChainEpoch {
	if a < b {
		return a
	}
	return b
}

var TerminationRewardFactor = builtin.BigFrac{ // PARAM_SPEC
	Numerator:   big.NewInt(1),
	Denominator: big.NewInt(2),
}

func PledgePenaltyForTermination(dayReward abi.TokenAmount, sectorAge abi.ChainEpoch,
	twentyDayRewardAtActivation abi.TokenAmount, networkQAPowerEstimate smoothing.FilterEstimate,
	qaSectorPower abi.StoragePower, rewardEstimate smoothing.FilterEstimate, replacedDayReward abi.TokenAmount,
	replacedSectorAge abi.ChainEpoch) abi.TokenAmount {
	// max(SP(t), BR(StartEpoch, 20d) + BR(StartEpoch, 1d) * terminationRewardFactor * min(SectorAgeInDays, 140))
	// and sectorAgeInDays = sectorAge / EpochsInDay
	lifetimeCap := abi.ChainEpoch(TerminationLifetimeCap) * builtin.EpochsInDay
	cappedSectorAge := minEpoch(sectorAge, lifetimeCap)
	// expected reward for lifetime of new sector (epochs*AttoFIL/day)
	expectedReward := big.Mul(dayReward, big.NewInt(int64(cappedSectorAge)))
	// if lifetime under cap and this sector replaced capacity, add expected reward for old sector's lifetime up to cap
	relevantReplacedAge := minEpoch(replacedSectorAge, lifetimeCap-cappedSectorAge)
	expectedReward = big.Add(expectedReward, big.Mul(replacedDayReward, big.NewInt(int64(relevantReplacedAge))))

	penalizedReward := big.Mul(expectedReward, TerminationRewardFactor.Numerator)

	return big.Max(
		PledgePenaltyForTerminationLowerBound(rewardEstimate, networkQAPowerEstimate, qaSectorPower),
		big.Add(
			twentyDayRewardAtActivation,
			big.Div(
				penalizedReward,
				big.Mul(big.NewInt(builtin.EpochsInDay), TerminationRewardFactor.Denominator)))) // (epochs*AttoFIL/day -> AttoFIL)
}

func PledgePenaltyForTerminationLowerBound(rewardEstimate, networkQAPowerEstimate smoothing.FilterEstimate, qaSectorPower abi.StoragePower) abi.TokenAmount {
	return ExpectedRewardForPower(rewardEstimate, networkQAPowerEstimate, qaSectorPower, TerminationPenaltyLowerBoundProjectionPeriod)
}

var TerminationPenaltyLowerBoundProjectionPeriod = abi.ChainEpoch((builtin.EpochsInDay * 35) / 10) // PARAM_SPEC

func ExpectedRewardForPower(rewardEstimate, networkQAPowerEstimate smoothing.FilterEstimate, qaSectorPower abi.StoragePower, projectionDuration abi.ChainEpoch) abi.TokenAmount {
	networkQAPowerSmoothed := smoothing.Estimate(&networkQAPowerEstimate)
	if networkQAPowerSmoothed.IsZero() {
		return smoothing.Estimate(&rewardEstimate)
	}
	expectedRewardForProvingPeriod := smoothing.ExtrapolatedCumSumOfRatio(projectionDuration, 0, rewardEstimate, networkQAPowerEstimate)
	br128 := big.Mul(qaSectorPower, expectedRewardForProvingPeriod) // Q.0 * Q.128 => Q.128
	br := big.Rsh(br128, math.Precision128)

	return big.Max(br, big.Zero())
}
