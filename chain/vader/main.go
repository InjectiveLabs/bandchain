package vader

import (
	"fmt"
	"os"
	"path"

	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/crypto/keys"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/version"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/bandprotocol/bandchain/chain/app"
)

const (
	flagRequester        = "requester"
	flagLogLevel         = "log-level"
	flagBroadcastTimeout = "broadcast-timeout"
	flagRPCPollInterval  = "rpc-poll-interval"
	flagMaxTry           = "max-try"
	flagOracleScriptID   = "oracle-script-id"
	flagAskCount         = "ask-count"
	flagMinCount         = "min-count"
	flagSymbols          = "symbols"
)

// Config data structure for vader daemon.
type Config struct {
	ChainID           string   `mapstructure:"chain-id"`            // ChainID of the target chain
	NodeURI           string   `mapstructure:"node"`                // Remote RPC URI of BandChain node to connect to
	Requester         string   `mapstructure:"requester"`           // The requester address that I'm responsible for
	OracleScriptID    int64    `mapstructure:"oracle-script-id"`    // The oracle script ID
	AskCount          uint64   `mapstructure:"ask-count"`           // The ask count
	MinCount          uint64   `mapstructure:"min-count"`           // The min count
	Symbols           []string `mapstructure:"symbols"`             // The symbols
	GasPrices         string   `mapstructure:"gas-prices"`          // Gas prices of the transaction
	LogLevel          string   `mapstructure:"log-level"`           // Log level of the logger
	BroadcastTimeout  string   `mapstructure:"broadcast-timeout"`   // The time that vader will wait for tx commit
	RPCPollInterval   string   `mapstructure:"rpc-poll-interval"`   // The duration of rpc poll interval
	MaxTry            uint64   `mapstructure:"max-try"`             // The maximum number of tries to submit a report transaction
	MetricsListenAddr string   `mapstructure:"metrics-listen-addr"` // Address to listen on for prometheus metrics
}

// Global instances.
var (
	cfg     Config
	keybase keys.Keybase
)

func initConfig(cmd *cobra.Command) error {
	home, err := cmd.PersistentFlags().GetString(flags.FlagHome)
	if err != nil {
		return err
	}
	viper.SetConfigFile(path.Join(home, "config.yaml"))
	_ = viper.ReadInConfig() // If we fail to read config file, we'll just rely on cmd flags.
	if err := viper.Unmarshal(&cfg); err != nil {
		return err
	}
	return nil
}

func Main() {
	appConfig := sdk.GetConfig()
	app.SetBech32AddressPrefixesAndBip44CoinType(appConfig)
	appConfig.Seal()

	ctx := &Context{}
	rootCmd := &cobra.Command{
		Use:   "vader",
		Short: "BandChain price requester daemon to create oracle requests",
	}

	rootCmd.AddCommand(configCmd(), keysCmd(ctx), runCmd(ctx), version.Cmd)
	rootCmd.PersistentPreRunE = func(_ *cobra.Command, _ []string) error {
		home, err := rootCmd.PersistentFlags().GetString(flags.FlagHome)
		if err != nil {
			return err
		}
		if err := os.MkdirAll(home, os.ModePerm); err != nil {
			return err
		}
		keybase, err = keys.NewKeyring("band", "test", home, nil)
		if err != nil {
			return err
		}
		return initConfig(rootCmd)
	}
	rootCmd.PersistentFlags().String(flags.FlagHome, os.ExpandEnv("$HOME/.vader"), "home directory")
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
