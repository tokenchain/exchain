package keeper

import (
	"fmt"

	"github.com/okex/exchain/libs/cosmos-sdk/codec"
	sdk "github.com/okex/exchain/libs/cosmos-sdk/types"
	"github.com/okex/exchain/x/debug/types"
)

// keeper of debug module
type Keeper struct {
	storeKey sdk.StoreKey
	cdc      *codec.Codec
	// for test for sending fee
	feePoolModuleName string
	orderKeeper       types.OrderKeeper
	StopFunc          func()
	stakingKeeper     types.StakingKeeper
	crisisKeeper      types.CrisisKeeper
}

func NewDebugKeeper(cdc *codec.Codec, storeKey sdk.StoreKey,
	orderKeeper types.OrderKeeper, stakingKeeper types.StakingKeeper, crisisKeeper types.CrisisKeeper,
	feePoolModuleName string, stop func()) Keeper {
	return Keeper{
		storeKey:          storeKey,
		cdc:               cdc,
		feePoolModuleName: feePoolModuleName,
		orderKeeper:       orderKeeper,
		StopFunc:          stop,
		stakingKeeper:     stakingKeeper,
		crisisKeeper:      crisisKeeper,
	}
}

func (k *Keeper) GetCDC() *codec.Codec {
	return k.cdc
}

func (k *Keeper) DumpStore(ctx sdk.Context, m string) {
	logger := ctx.Logger().With("module", "debug")

	logger.Error("--------------------------------------------------------------------------")
	logger.Error(fmt.Sprintf("---------- Dump <%s> KV Store at BlockHeight <%d> started ----------",
		m, ctx.BlockHeight()))
	defer logger.Error("--------------------------------------------------------------------------")
	defer logger.Error(fmt.Sprintf("---------- Dump <%s> KV Store at BlockHeight <%d> finished ----------",
		m, ctx.BlockHeight()))

	if m == "order" {
		k.orderKeeper.DumpStore(ctx)
	}
}
