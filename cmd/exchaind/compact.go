package main

import (
	"fmt"
	"log"
	"sync"

	"github.com/syndtr/goleveldb/leveldb/util"
	dbm "github.com/tendermint/tm-db"

	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/server"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/tendermint/tendermint/node"
)

var wg sync.WaitGroup

func compactCmd(ctx *server.Context) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "compact [all|app|block|state]",
		Short: "Compact the leveldb",
		Long: `all   : compact all of application, blockstore and state db
app   : only compact application db
block : only compact blockstore db
state : only compact state db
`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				return fmt.Errorf("invalid args")
			}

			config := ctx.Config
			config.SetRoot(viper.GetString(flags.FlagHome))
			blockStoreDB, stateDB, appDB, err := initDBs(config, node.DefaultDBProvider)
			if err != nil {
				return err
			}

			switch args[0] {
			case "app":
				log.Println("--------- compact app start ---------")
				wg.Add(1)
				go compactDB(appDB)
				wg.Wait()
				log.Println("--------- compact app end ---------")
			case "block":
				log.Println("--------- compact block start ---------")
				wg.Add(1)
				go compactDB(blockStoreDB)
				wg.Wait()
				log.Println("--------- compact block end ---------")
			case "state":
				log.Println("--------- compact state start ---------")
				wg.Add(1)
				go compactDB(stateDB)
				wg.Wait()
				log.Println("--------- compact state end ---------")
				return nil
			case "all":
				log.Println("--------- compact all start ---------")
				wg.Add(3)
				go compactDB(blockStoreDB)
				go compactDB(stateDB)
				go compactDB(appDB)
				wg.Wait()
				log.Println("--------- compact all end ---------")
			}
			return nil
		},
	}

	return cmd
}

func compactDB(db dbm.DB) {
	defer wg.Done()
	err := db.(*dbm.GoLevelDB).DB().CompactRange(util.Range{})
	panicError(err)
}
