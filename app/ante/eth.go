package ante

import (
	"fmt"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"math/big"

	"github.com/okex/exchain/libs/cosmos-sdk/baseapp"

	sdk "github.com/okex/exchain/libs/cosmos-sdk/types"
	sdkerrors "github.com/okex/exchain/libs/cosmos-sdk/types/errors"
	"github.com/okex/exchain/libs/cosmos-sdk/x/auth"
	authante "github.com/okex/exchain/libs/cosmos-sdk/x/auth/ante"
	"github.com/okex/exchain/libs/cosmos-sdk/x/auth/types"
	"github.com/ethereum/go-ethereum/common"
	ethcore "github.com/ethereum/go-ethereum/core"
	ethermint "github.com/okex/exchain/app/types"
	evmtypes "github.com/okex/exchain/x/evm/types"
)

// EVMKeeper defines the expected keeper interface used on the Eth AnteHandler
type EVMKeeper interface {
	GetParams(ctx sdk.Context) evmtypes.Params
	IsAddressBlocked(ctx sdk.Context, addr sdk.AccAddress) bool
}

// EthSetupContextDecorator sets the infinite GasMeter in the Context and wraps
// the next AnteHandler with a defer clause to recover from any downstream
// OutOfGas panics in the AnteHandler chain to return an error with information
// on gas provided and gas used.
// CONTRACT: Must be first decorator in the chain
// CONTRACT: Tx must implement GasTx interface
type EthSetupContextDecorator struct{}

// NewEthSetupContextDecorator creates a new EthSetupContextDecorator
func NewEthSetupContextDecorator() EthSetupContextDecorator {
	return EthSetupContextDecorator{}
}

// AnteHandle sets the infinite gas meter to done to ignore costs in AnteHandler checks.
// This is undone at the EthGasConsumeDecorator, where the context is set with the
// ethereum tx GasLimit.
func (escd EthSetupContextDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (newCtx sdk.Context, err error) {
	// all transactions must implement GasTx
	gasTx, ok := tx.(authante.GasTx)
	if !ok {
		return ctx, sdkerrors.Wrap(sdkerrors.ErrTxDecode, "Tx must be GasTx")
	}

	// Decorator will catch an OutOfGasPanic caused in the next antehandler
	// AnteHandlers must have their own defer/recover in order for the BaseApp
	// to know how much gas was used! This is because the GasMeter is created in
	// the AnteHandler, but if it panics the context won't be set properly in
	// runTx's recover call.
	defer func() {
		if r := recover(); r != nil {
			switch rType := r.(type) {
			case sdk.ErrorOutOfGas:
				log := fmt.Sprintf(
					"out of gas in location: %v; gasLimit: %d, gasUsed: %d",
					rType.Descriptor, gasTx.GetGas(), ctx.GasMeter().GasConsumed(),
				)
				err = sdkerrors.Wrap(sdkerrors.ErrOutOfGas, log)
			default:
				panic(r)
			}
		}
	}()

	return next(ctx, tx, simulate)
}

// EthMempoolFeeDecorator validates that sufficient fees have been provided that
// meet a minimum threshold defined by the proposer (for mempool purposes during CheckTx).
type EthMempoolFeeDecorator struct {
	evmKeeper EVMKeeper
}

// NewEthMempoolFeeDecorator creates a new EthMempoolFeeDecorator
func NewEthMempoolFeeDecorator(ek EVMKeeper) EthMempoolFeeDecorator {
	return EthMempoolFeeDecorator{
		evmKeeper: ek,
	}
}

// AnteHandle verifies that enough fees have been provided by the
// Ethereum transaction that meet the minimum threshold set by the block
// proposer.
//
// NOTE: This should only be run during a CheckTx mode.
func (emfd EthMempoolFeeDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (newCtx sdk.Context, err error) {
	if !ctx.IsCheckTx() {
		return next(ctx, tx, simulate)
	}

	msgEthTx, ok := tx.(evmtypes.MsgEthereumTx)
	if !ok {
		return ctx, sdkerrors.Wrapf(sdkerrors.ErrUnknownRequest, "invalid transaction type: %T", tx)
	}

	evmDenom := sdk.DefaultBondDenom

	// fee = gas price * gas limit
	fee := sdk.NewDecCoinFromDec(evmDenom, sdk.NewDecFromBigIntWithPrec(msgEthTx.Fee(), sdk.Precision))

	minGasPrices := ctx.MinGasPrices()
	minFees := minGasPrices.AmountOf(evmDenom).MulInt64(int64(msgEthTx.Data.GasLimit))

	// check that fee provided is greater than the minimum defined by the validator node
	// NOTE: we only check if the evm denom tokens are present in min gas prices. It is up to the
	// sender if they want to send additional fees in other denominations.
	var hasEnoughFees bool
	if fee.Amount.GTE(minFees) {
		hasEnoughFees = true
	}

	// reject transaction if minimum gas price is not zero and the transaction does not
	// meet the minimum fee
	if !ctx.MinGasPrices().IsZero() && !hasEnoughFees {
		return ctx, sdkerrors.Wrap(
			sdkerrors.ErrInsufficientFee,
			fmt.Sprintf("insufficient fee, got: %q required: %q", fee, sdk.NewDecCoinFromDec(evmDenom, minFees)),
		)
	}

	return next(ctx, tx, simulate)
}

// EthSigVerificationDecorator validates an ethereum signature
type EthSigVerificationDecorator struct{}

// NewEthSigVerificationDecorator creates a new EthSigVerificationDecorator
func NewEthSigVerificationDecorator() EthSigVerificationDecorator {
	return EthSigVerificationDecorator{}
}

// AnteHandle validates the signature and returns sender address
func (esvd EthSigVerificationDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (newCtx sdk.Context, err error) {
	msgEthTx, ok := tx.(evmtypes.MsgEthereumTx)
	if !ok {
		return ctx, sdkerrors.Wrapf(sdkerrors.ErrUnknownRequest, "invalid transaction type: %T", tx)
	}

	// parse the chainID from a string to a base-10 integer
	chainIDEpoch, err := ethermint.ParseChainID(ctx.ChainID())
	if err != nil {
		return ctx, err
	}

	// validate sender/signature and cache the address
	signerSigCache, err := msgEthTx.VerifySig(chainIDEpoch, ctx.BlockHeight(), ctx.SigCache())
	if err != nil {
		return ctx, sdkerrors.Wrapf(sdkerrors.ErrUnauthorized, "signature verification failed: %s", err.Error())
	}

	// update ctx for push signerSigCache
	newCtx = ctx.WithSigCache(signerSigCache)

	// NOTE: when signature verification succeeds, a non-empty signer address can be
	// retrieved from the transaction on the next AnteDecorators.
	return next(newCtx, msgEthTx, simulate)
}

// AccountVerificationDecorator validates an account balance checks
type AccountVerificationDecorator struct {
	ak        auth.AccountKeeper
	evmKeeper EVMKeeper
}

// NewAccountVerificationDecorator creates a new AccountVerificationDecorator
func NewAccountVerificationDecorator(ak auth.AccountKeeper, ek EVMKeeper) AccountVerificationDecorator {
	return AccountVerificationDecorator{
		ak:        ak,
		evmKeeper: ek,
	}
}

// AnteHandle validates the signature and returns sender address
func (avd AccountVerificationDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (newCtx sdk.Context, err error) {
	if !ctx.IsCheckTx() {
		return next(ctx, tx, simulate)
	}

	msgEthTx, ok := tx.(evmtypes.MsgEthereumTx)
	if !ok {
		return ctx, sdkerrors.Wrapf(sdkerrors.ErrUnknownRequest, "invalid transaction type: %T", tx)
	}

	// sender address should be in the tx cache from the previous AnteHandle call
	address := msgEthTx.From()
	if address.Empty() {
		panic("sender address cannot be empty")
	}

	acc := avd.ak.GetAccount(ctx, address)
	if acc == nil {
		acc = avd.ak.NewAccountWithAddress(ctx, address)
		avd.ak.SetAccount(ctx, acc)
	}

	// on InitChain make sure account number == 0
	if ctx.BlockHeight() == 0 && acc.GetAccountNumber() != 0 {
		return ctx, sdkerrors.Wrapf(
			sdkerrors.ErrInvalidSequence,
			"invalid account number for height zero (got %d)", acc.GetAccountNumber(),
		)
	}

	evmDenom := sdk.DefaultBondDenom

	// validate sender has enough funds to pay for gas cost
	balance := acc.GetCoins().AmountOf(evmDenom)
	if balance.BigInt().Cmp(msgEthTx.Cost()) < 0 {
		return ctx, sdkerrors.Wrapf(
			sdkerrors.ErrInsufficientFunds,
			"sender balance < tx gas cost (%s%s < %s%s)", balance.String(), evmDenom, sdk.NewDecFromBigIntWithPrec(msgEthTx.Cost(), sdk.Precision).String(), evmDenom,
		)
	}

	return next(ctx, tx, simulate)
}

// NonceVerificationDecorator checks that the account nonce from the transaction matches
// the sender account sequence.
type NonceVerificationDecorator struct {
	ak auth.AccountKeeper
}

// NewNonceVerificationDecorator creates a new NonceVerificationDecorator
func NewNonceVerificationDecorator(ak auth.AccountKeeper) NonceVerificationDecorator {
	return NonceVerificationDecorator{
		ak: ak,
	}
}

// AnteHandle validates that the transaction nonce is valid (equivalent to the sender account’s
// current nonce).
func (nvd NonceVerificationDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (newCtx sdk.Context, err error) {
	msgEthTx, ok := tx.(evmtypes.MsgEthereumTx)
	if !ok {
		return ctx, sdkerrors.Wrapf(sdkerrors.ErrUnknownRequest, "invalid transaction type: %T", tx)
	}

	// sender address should be in the tx cache from the previous AnteHandle call
	address := msgEthTx.From()
	if address.Empty() {
		panic("sender address cannot be empty")
	}

	acc := nvd.ak.GetAccount(ctx, address)
	if acc == nil {
		return ctx, sdkerrors.Wrapf(
			sdkerrors.ErrUnknownAddress,
			"account %s (%s) is nil", common.BytesToAddress(address.Bytes()), address,
		)
	}

	seq := acc.GetSequence()
	// if multiple transactions are submitted in succession with increasing nonces,
	// all will be rejected except the first, since the first needs to be included in a block
	// before the sequence increments
	if ctx.IsCheckTx() {
		ctx = ctx.WithAccountNonce(seq)
		// will be checkTx and RecheckTx mode
		if ctx.IsReCheckTx() {
			// recheckTx mode

			// sequence must strictly increasing
			if msgEthTx.Data.AccountNonce != seq {
				return ctx, sdkerrors.Wrapf(
					sdkerrors.ErrInvalidSequence,
					"invalid nonce; got %d, expected %d", msgEthTx.Data.AccountNonce, seq,
				)
			}
		} else {
			if baseapp.IsMempoolEnablePendingPool() {
				if msgEthTx.Data.AccountNonce < seq {
					return ctx, sdkerrors.Wrapf(
						sdkerrors.ErrInvalidSequence,
						"invalid nonce; got %d, expected %d",
						msgEthTx.Data.AccountNonce, seq,
					)
				}
			} else {
				// checkTx mode
				checkTxModeNonce := seq

				if !baseapp.IsMempoolEnableRecheck() {
					// if is enable recheck, the sequence of checkState will increase after commit(), so we do not need
					// to add pending txs len in the mempool.
					// but, if disable recheck, we will not increase sequence of checkState (even in force recheck case, we
					// will also reset checkState), so we will need to add pending txs len to get the right nonce
					gPool := baseapp.GetGlobalMempool()
					if gPool != nil {
						cnt := gPool.GetUserPendingTxsCnt(common.BytesToAddress(address.Bytes()).String())
						checkTxModeNonce = seq + uint64(cnt)
					}
				}

				if baseapp.IsMempoolEnableSort() {
					if msgEthTx.Data.AccountNonce < seq || msgEthTx.Data.AccountNonce > checkTxModeNonce {
						return ctx, sdkerrors.Wrapf(
							sdkerrors.ErrInvalidSequence,
							"invalid nonce; got %d, expected in the range of [%d, %d]",
							msgEthTx.Data.AccountNonce, seq, checkTxModeNonce,
						)
					}
				} else {
					if msgEthTx.Data.AccountNonce != checkTxModeNonce {
						return ctx, sdkerrors.Wrapf(
							sdkerrors.ErrInvalidSequence,
							"invalid nonce; got %d, expected %d",
							msgEthTx.Data.AccountNonce, checkTxModeNonce,
						)
					}
				}
			}
		}
	} else {
		// only deliverTx mode
		if msgEthTx.Data.AccountNonce != seq {
			return ctx, sdkerrors.Wrapf(
				sdkerrors.ErrInvalidSequence,
				"invalid nonce; got %d, expected %d", msgEthTx.Data.AccountNonce, seq,
			)
		}
	}

	return next(ctx, tx, simulate)
}

// EthGasConsumeDecorator validates enough intrinsic gas for the transaction and
// gas consumption.
type EthGasConsumeDecorator struct {
	ak        auth.AccountKeeper
	sk        types.SupplyKeeper
	evmKeeper EVMKeeper
}

// NewEthGasConsumeDecorator creates a new EthGasConsumeDecorator
func NewEthGasConsumeDecorator(ak auth.AccountKeeper, sk types.SupplyKeeper, ek EVMKeeper) EthGasConsumeDecorator {
	return EthGasConsumeDecorator{
		ak:        ak,
		sk:        sk,
		evmKeeper: ek,
	}
}

// AnteHandle validates that the Ethereum tx message has enough to cover intrinsic gas
// (during CheckTx only) and that the sender has enough balance to pay for the gas cost.
//
// Intrinsic gas for a transaction is the amount of gas
// that the transaction uses before the transaction is executed. The gas is a
// constant value of 21000 plus any cost inccured by additional bytes of data
// supplied with the transaction.
func (egcd EthGasConsumeDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (newCtx sdk.Context, err error) {
	msgEthTx, ok := tx.(evmtypes.MsgEthereumTx)
	if !ok {
		return ctx, sdkerrors.Wrapf(sdkerrors.ErrUnknownRequest, "invalid transaction type: %T", tx)
	}

	// sender address should be in the tx cache from the previous AnteHandle call
	address := msgEthTx.From()
	if address.Empty() {
		panic("sender address cannot be empty")
	}

	// fetch sender account from signature
	senderAcc, err := auth.GetSignerAcc(ctx, egcd.ak, address)
	if err != nil {
		return ctx, err
	}

	if senderAcc == nil {
		return ctx, sdkerrors.Wrapf(
			sdkerrors.ErrUnknownAddress,
			"sender account %s (%s) is nil", common.BytesToAddress(address.Bytes()), address,
		)
	}

	gasLimit := msgEthTx.GetGas()
	gas, err := ethcore.IntrinsicGas(msgEthTx.Data.Payload, []ethtypes.AccessTuple{}, msgEthTx.To() == nil, true, false)
	if err != nil {
		return ctx, sdkerrors.Wrap(err, "failed to compute intrinsic gas cost")
	}

	// intrinsic gas verification during CheckTx
	if ctx.IsCheckTx() && gasLimit < gas {
		return ctx, sdkerrors.Wrapf(sdkerrors.ErrOutOfGas, "intrinsic gas too low: %d < %d", gasLimit, gas)
	}

	// Charge sender for gas up to limit
	if gasLimit != 0 {
		// Cost calculates the fees paid to validators based on gas limit and price
		cost := new(big.Int).Mul(msgEthTx.Data.Price, new(big.Int).SetUint64(gasLimit))

		evmDenom := sdk.DefaultBondDenom

		feeAmt := sdk.NewCoins(
			sdk.NewCoin(evmDenom, sdk.NewDecFromBigIntWithPrec(cost, sdk.Precision)), // int2dec
		)

		err = auth.DeductFees(egcd.sk, ctx, senderAcc, feeAmt)
		if err != nil {
			return ctx, err
		}
	}

	// Set gas meter after ante handler to ignore gaskv costs
	newCtx = auth.SetGasMeter(simulate, ctx, gasLimit)
	return next(newCtx, tx, simulate)
}

// IncrementSenderSequenceDecorator increments the sequence of the signers. The
// main difference with the SDK's IncrementSequenceDecorator is that the MsgEthereumTx
// doesn't implement the SigVerifiableTx interface.
//
// CONTRACT: must be called after msg.VerifySig in order to cache the sender address.
type IncrementSenderSequenceDecorator struct {
	ak auth.AccountKeeper
}

// NewIncrementSenderSequenceDecorator creates a new IncrementSenderSequenceDecorator.
func NewIncrementSenderSequenceDecorator(ak auth.AccountKeeper) IncrementSenderSequenceDecorator {
	return IncrementSenderSequenceDecorator{
		ak: ak,
	}
}

// AnteHandle handles incrementing the sequence of the sender.
func (issd IncrementSenderSequenceDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (sdk.Context, error) {
	// always incrementing the sequence when ctx is recheckTx mode (when mempool in disableRecheck mode, we will also has force recheck),
	// when mempool is in enableRecheck mode, we will need to increase the nonce when ctx is checkTx mode
	// when mempool is not in enableRecheck mode, we should not increment the nonce

	// when IsCheckTx() is true, it will means checkTx and recheckTx mode, but IsReCheckTx() is true it must be recheckTx mode
	if ctx.IsCheckTx() && !ctx.IsReCheckTx() && !baseapp.IsMempoolEnableRecheck() {
		return next(ctx, tx, simulate)
	}

	// get and set account must be called with an infinite gas meter in order to prevent
	// additional gas from being deducted.
	gasMeter := ctx.GasMeter()
	ctx = ctx.WithGasMeter(sdk.NewInfiniteGasMeter())

	msgEthTx, ok := tx.(evmtypes.MsgEthereumTx)
	if !ok {
		ctx = ctx.WithGasMeter(gasMeter)
		return ctx, sdkerrors.Wrapf(sdkerrors.ErrUnknownRequest, "invalid transaction type: %T", tx)
	}

	// increment sequence of all signers
	for _, addr := range msgEthTx.GetSigners() {
		acc := issd.ak.GetAccount(ctx, addr)
		seq := acc.GetSequence()
		if !baseapp.IsMempoolEnablePendingPool() {
			seq++
		} else if msgEthTx.Data.AccountNonce == seq {
			seq++
		}
		if err := acc.SetSequence(seq); err != nil {
			panic(err)
		}
		issd.ak.SetAccount(ctx, acc)
	}

	// set the original gas meter
	ctx = ctx.WithGasMeter(gasMeter)
	return next(ctx, tx, simulate)
}

// NewGasLimitDecorator creates a new GasLimitDecorator.
func NewGasLimitDecorator(evm EVMKeeper) GasLimitDecorator {
	return GasLimitDecorator{
		evm: evm,
	}
}

type GasLimitDecorator struct {
	evm EVMKeeper
}

// AnteHandle handles incrementing the sequence of the sender.
func (g GasLimitDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (sdk.Context, error) {
	msgEthTx, ok := tx.(evmtypes.MsgEthereumTx)
	if !ok {
		return ctx, sdkerrors.Wrapf(sdkerrors.ErrUnknownRequest, "invalid transaction type: %T", tx)
	}
	if msgEthTx.GetGas() > g.evm.GetParams(ctx).MaxGasLimitPerTx {
		return ctx, sdkerrors.Wrapf(sdkerrors.ErrTxTooLarge, "too large gas limit, it must be less than %d", g.evm.GetParams(ctx).MaxGasLimitPerTx)
	}
	return next(ctx, tx, simulate)
}
