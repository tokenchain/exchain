package core

import (
	"github.com/okex/exchain/libs/tendermint/evidence"
	ctypes "github.com/okex/exchain/libs/tendermint/rpc/core/types"
	rpctypes "github.com/okex/exchain/libs/tendermint/rpc/jsonrpc/types"
	"github.com/okex/exchain/libs/tendermint/types"
)

// BroadcastEvidence broadcasts evidence of the misbehavior.
// More: https://docs.tendermint.com/master/rpc/#/Info/broadcast_evidence
func BroadcastEvidence(ctx *rpctypes.Context, ev types.Evidence) (*ctypes.ResultBroadcastEvidence, error) {
	err := env.EvidencePool.AddEvidence(ev)
	if _, ok := err.(evidence.ErrEvidenceAlreadyStored); err == nil || ok {
		return &ctypes.ResultBroadcastEvidence{Hash: ev.Hash()}, nil
	}
	return nil, err
}
