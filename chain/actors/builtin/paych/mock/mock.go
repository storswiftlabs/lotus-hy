package mock

import (
	"io"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/big"
	"github.com/filecoin-project/lotus/chain/actors/builtin/paych"
)

type mockState struct {
	from       address.Address
	to         address.Address
	settlingAt abi.ChainEpoch
	toSend     abi.TokenAmount
	lanes      map[uint64]paych.LaneState
}

type mockLaneState struct {
	redeemed big.Int
	nonce    uint64
}

// NewMockPayChState constructs a state for a payment channel with the set fixed values
// that satisfies the paych.State interface.
func NewMockPayChState(from address.Address,
	to address.Address,
	settlingAt abi.ChainEpoch,
	toSend abi.TokenAmount,
	lanes map[uint64]paych.LaneState,
) paych.State {
	return &mockState{from, to, settlingAt, toSend, lanes}
}

// NewMockLaneState constructs a state for a payment channel lane with the set fixed values
// that satisfies the paych.LaneState interface. Useful for populating lanes when
// calling NewMockPayChState
func NewMockLaneState(redeemed big.Int, nonce uint64) paych.LaneState {
	return &mockLaneState{redeemed, nonce}
}

func (ms *mockState) MarshalCBOR(io.Writer) error {
	panic("not implemented")
}

// Channel owner, who has funded the actor
func (ms *mockState) From() address.Address {
	return ms.from
}

// Recipient of payouts from channel
func (ms *mockState) To() address.Address {
	return ms.to
}

// Height at which the channel can be `Collected`
func (ms *mockState) SettlingAt() abi.ChainEpoch {
	return ms.settlingAt
}

// Amount successfully redeemed through the payment channel, paid out on `Collect()`
func (ms *mockState) ToSend() abi.TokenAmount {
	return ms.toSend
}

// Get total number of lanes
func (ms *mockState) LaneCount() (uint64, error) {
	return uint64(len(ms.lanes)), nil
}

// Iterate lane states
func (ms *mockState) ForEachLaneState(cb func(idx uint64, dl paych.LaneState) error) error {
	var lastErr error
	for lane, state := range ms.lanes {
		if err := cb(lane, state); err != nil {
			lastErr = err
		}
	}
	return lastErr
}

func (mls *mockLaneState) Redeemed() big.Int {
	return mls.redeemed
}

func (mls *mockLaneState) Nonce() uint64 {
	return mls.nonce
}
