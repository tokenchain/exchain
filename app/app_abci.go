package app

import (
	appconfig "github.com/okex/exchain/app/config"
	"github.com/okex/exchain/x/common/analyzer"
	"github.com/okex/exchain/x/evm"
	abci "github.com/okex/exchain/libs/tendermint/abci/types"
	"github.com/okex/exchain/libs/tendermint/trace"
)

// BeginBlock implements the Application interface
func (app *OKExChainApp) BeginBlock(req abci.RequestBeginBlock) (res abci.ResponseBeginBlock) {

	analyzer.OnAppBeginBlockEnter(app.LastBlockHeight() + 1)
	defer analyzer.OnAppBeginBlockExit()

	// dump app.LastBlockHeight()-1 info for reactor sync mode
	trace.GetElapsedInfo().Dump(app.Logger())
	return app.BaseApp.BeginBlock(req)
}

func (app *OKExChainApp) DeliverTx(req abci.RequestDeliverTx) (res abci.ResponseDeliverTx) {

	analyzer.OnAppDeliverTxEnter()
	defer analyzer.OnAppDeliverTxExit()

	resp := app.BaseApp.DeliverTx(req)

	if appconfig.GetOecConfig().GetEnableDynamicGp() {
		tx, err := evm.TxDecoder(app.Codec())(req.Tx)
		if err == nil {
			//optimize get tx gas price can not get value from verifySign method
			app.blockGasPrice = append(app.blockGasPrice, tx.GetGasPrice())
		}
	}

	return resp
}

// EndBlock implements the Application interface
func (app *OKExChainApp) EndBlock(req abci.RequestEndBlock) (res abci.ResponseEndBlock) {

	analyzer.OnAppEndBlockEnter()
	defer analyzer.OnAppEndBlockExit()

	return app.BaseApp.EndBlock(req)
}

// Commit implements the Application interface
func (app *OKExChainApp) Commit() abci.ResponseCommit {

	analyzer.OnCommitEnter()
	defer analyzer.OnCommitExit()
	res := app.BaseApp.Commit()

	return res
}
