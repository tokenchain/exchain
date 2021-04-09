package main

import (
	"github.com/cosmos/cosmos-sdk/store/rootmulti"
	"github.com/okex/okexchain/app"
	"log"
	"path/filepath"

	"github.com/cosmos/cosmos-sdk/server"
	"github.com/spf13/cobra"
)

func exportAppCmd(ctx *server.Context) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "exportApp",
		Short: "export current snapshot to new db",
		Run: func(cmd *cobra.Command, args []string) {
			log.Println("--------- export start ---------")
			export(ctx)
			log.Println("--------- export success ---------")
		},
	}
	return cmd
}

func export(ctx *server.Context) {
	fromApp := createApp(ctx, "data")
	toApp := createApp(ctx, "export")

	fromRs, _ := fromApp.BaseApp.GetCms().(*rootmulti.Store)
	toRs := toApp.BaseApp.GetCms().(*rootmulti.Store)

	version := fromRs.LastCommitID().Version
	log.Println("export app version ", version)

	err := fromRs.Export(toRs, version-1)
	if err != nil {
		panicError(err)
	}
}

func createApp(ctx *server.Context, dataPath string) *app.OKExChainApp {
	rootDir := ctx.Config.RootDir
	dataDir := filepath.Join(rootDir, dataPath)
	db, err := openDB(applicationDB, dataDir)
	panicError(err)
	exapp := newApp(ctx.Logger, db, nil)
	return exapp.(*app.OKExChainApp)
}
