package main

import (
	"github.com/cosmos/cosmos-sdk/server"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/syndtr/goleveldb/leveldb/util"
	dbm "github.com/tendermint/tm-db"
	"log"
	"path/filepath"
)

func compactCmd(ctx *server.Context) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "compact",
		Short: "Compact db",
		Run: func(cmd *cobra.Command, args []string) {
			log.Println("--------- compact start ---------")
			dataDir := viper.GetString(dataDirFlag)
			compactDB(ctx, dataDir)
			log.Println("--------- compact success ---------")
		},
	}
	cmd.Flags().StringP(dataDirFlag, "d", ".okexchaind/data", "Directory of block data for replaying")
	return cmd
}

func compactDB(ctx *server.Context, dir string) {
	rootDir := ctx.Config.RootDir
	dataDir := filepath.Join(rootDir, "data")
	db, err := openDB(applicationDB, dataDir)
	panicError(err)
	err = db.(*dbm.GoLevelDB).DB().CompactRange(util.Range{nil, nil})
	panicError(err)
}
