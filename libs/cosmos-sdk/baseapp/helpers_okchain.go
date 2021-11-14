package baseapp

import (
	sdk "github.com/okex/exchain/libs/cosmos-sdk/types"
)

func (app *BaseApp) PushAnteHandler(ah sdk.AnteHandler) {
	app.anteHandler = ah
}

func (app *BaseApp) GetDeliverStateCtx() sdk.Context {
	return app.deliverState.ctx
}