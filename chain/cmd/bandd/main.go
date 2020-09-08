package main

import (
	"encoding/json"
	"io"

	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/server"
	"github.com/cosmos/cosmos-sdk/store"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/genaccounts"
	genaccscli "github.com/cosmos/cosmos-sdk/x/genaccounts/client/cli"
	genutilcli "github.com/cosmos/cosmos-sdk/x/genutil/client/cli"
	"github.com/cosmos/cosmos-sdk/x/staking"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	abci "github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/libs/cli"
	"github.com/tendermint/tendermint/libs/log"
	tmtypes "github.com/tendermint/tendermint/types"
	dbm "github.com/tendermint/tm-db"

	"github.com/bandprotocol/bandchain/chain/app"
	replay "github.com/bandprotocol/bandchain/chain/emitter/replay"
	sync "github.com/bandprotocol/bandchain/chain/emitter/sync"
)

const (
	flagInvCheckPeriod        = "inv-check-period"
	flagWithEmitter           = "with-emitter"
	flagDisableFeelessReports = "disable-feeless-reports"
	flagEnableFastSync        = "enable-fast-sync"
	flagReplayMode            = "replay-mode"
)

var (
	invCheckPeriod uint
)

func main() {
	cobra.EnableCommandSorting = false

	cdc := app.MakeCodec()

	config := sdk.GetConfig()
	app.SetBech32AddressPrefixesAndBip44CoinType(config)
	config.Seal()

	ctx := server.NewDefaultContext()

	rootCmd := &cobra.Command{
		Use:               "bandd",
		Short:             "BandChain App Daemon (server)",
		PersistentPreRunE: server.PersistentPreRunEFn(ctx),
	}
	// CLI commands to initialize the chain
	rootCmd.AddCommand(
		InitCmd(ctx, cdc, app.NewDefaultGenesisState() /* app.GetDefaultDataSourcesAndOracleScripts, */, app.DefaultNodeHome),
		genutilcli.CollectGenTxsCmd(ctx, cdc, genaccounts.AppModuleBasic{}, app.DefaultNodeHome),
		genutilcli.MigrateGenesisCmd(ctx, cdc),
		genutilcli.GenTxCmd(
			ctx, cdc, app.ModuleBasics, staking.AppModuleBasic{},
			genaccounts.AppModuleBasic{}, app.DefaultNodeHome, app.DefaultCLIHome,
		),
		genutilcli.ValidateGenesisCmd(ctx, cdc, app.ModuleBasics),
		genaccscli.AddGenesisAccountCmd(ctx, cdc, app.DefaultNodeHome, app.DefaultCLIHome),
		client.NewCompletionCmd(rootCmd, true),
	)

	server.AddCommands(ctx, cdc, rootCmd, newApp, exportAppStateAndTMValidators)

	// prepare and add flags
	executor := cli.PrepareBaseCmd(rootCmd, "BAND", app.DefaultNodeHome)
	rootCmd.PersistentFlags().UintVar(&invCheckPeriod, flagInvCheckPeriod, 0, "Assert registered invariants every N blocks")
	rootCmd.PersistentFlags().String(flagWithEmitter, "", "[Experimental] Use Kafka emitter")
	rootCmd.PersistentFlags().Bool(flagEnableFastSync, false, "[Experimental] Enable fast sync mode")
	rootCmd.PersistentFlags().Bool(flagReplayMode, false, "[Experimental] Use emitter replay mode")
	rootCmd.PersistentFlags().Bool(flagDisableFeelessReports, false, "[Experimental] Disable allowance of feeless reports")
	err := executor.Execute()
	if err != nil {
		panic(err)
	}
}

func newApp(logger log.Logger, db dbm.DB, traceStore io.Writer) abci.Application {
	var cache sdk.MultiStorePersistentCache

	if viper.GetBool(server.FlagInterBlockCache) {
		cache = store.NewCommitKVStoreCacheManager()
	}

	skipUpgradeHeights := make(map[int64]bool)
	for _, h := range viper.GetIntSlice(server.FlagUnsafeSkipUpgrades) {
		skipUpgradeHeights[int64(h)] = true
	}
	pruningOpts, err := server.GetPruningOptionsFromFlags()
	if err != nil {
		panic(err)
	}
	if viper.IsSet(flagWithEmitter) {
		if viper.GetBool(flagReplayMode) {
			return replay.NewBandAppWithEmitter(
				viper.GetString(flagWithEmitter), logger, db, traceStore, true, invCheckPeriod,
				skipUpgradeHeights, viper.GetString(flags.FlagHome),
				viper.GetBool(flagDisableFeelessReports),
				baseapp.SetPruning(pruningOpts),
				baseapp.SetMinGasPrices(viper.GetString(server.FlagMinGasPrices)),
				baseapp.SetHaltHeight(viper.GetUint64(server.FlagHaltHeight)),
				baseapp.SetHaltTime(viper.GetUint64(server.FlagHaltTime)),
				baseapp.SetInterBlockCache(cache),
			)
		} else {
			return sync.NewBandAppWithEmitter(
				viper.GetString(flagWithEmitter), logger, db, traceStore, true, invCheckPeriod,
				skipUpgradeHeights, viper.GetString(flags.FlagHome),
				viper.GetBool(flagDisableFeelessReports),
				viper.GetBool(flagEnableFastSync),
				baseapp.SetPruning(pruningOpts),
				baseapp.SetMinGasPrices(viper.GetString(server.FlagMinGasPrices)),
				baseapp.SetHaltHeight(viper.GetUint64(server.FlagHaltHeight)),
				baseapp.SetHaltTime(viper.GetUint64(server.FlagHaltTime)),
				baseapp.SetInterBlockCache(cache),
			)
		}
	} else {
		return app.NewBandApp(
			logger, db, traceStore, true, invCheckPeriod,
			baseapp.SetPruning(store.NewPruningOptionsFromString(viper.GetString("pruning"))),
			baseapp.SetMinGasPrices(viper.GetString(server.FlagMinGasPrices)),
			baseapp.SetHaltHeight(uint64(viper.GetInt(server.FlagHaltHeight))))
	}
}

func exportAppStateAndTMValidators(
	logger log.Logger, db dbm.DB, traceStore io.Writer, height int64, forZeroHeight bool, jailWhiteList []string,
) (json.RawMessage, []tmtypes.GenesisValidator, error) {

	if height != -1 {
		bandApp := app.NewBandApp(logger, db, traceStore, false, uint(1))
		err := bandApp.LoadHeight(height)
		if err != nil {
			return nil, nil, err
		}
		return bandApp.ExportAppStateAndValidators(forZeroHeight, jailWhiteList)
	}

	bandApp := app.NewBandApp(logger, db, traceStore, true, uint(1))
	return bandApp.ExportAppStateAndValidators(forZeroHeight, jailWhiteList)
}
