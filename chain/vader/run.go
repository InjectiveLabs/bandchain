package vader

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"time"

	"github.com/cosmos/cosmos-sdk/client/flags"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/tendermint/tendermint/libs/log"
	httpclient "github.com/tendermint/tendermint/rpc/client/http"

	pricetypes "github.com/bandprotocol/bandchain/chain/hooks/price"
	"github.com/bandprotocol/bandchain/chain/pkg/filecache"
	"github.com/bandprotocol/bandchain/chain/pkg/obi"
	oracletypes "github.com/bandprotocol/bandchain/chain/x/oracle/types"
)

const BandPriceMultiplier uint64 = 1000000000 // 1e9

func runImpl(c *Context, l *Logger) error {
	l.Info(":rocket: Starting WebSocket subscriber")
	err := c.client.Start()
	if err != nil {
		return err
	}

	ctx, cxl := context.WithTimeout(context.Background(), 5*time.Second)
	defer cxl()

	if c.metricsEnabled {
		l.Info(":eyes: Starting Prometheus listener")
		go metricsListen(cfg.MetricsListenAddr, c)
	}

	for {
		fmt.Println("Hey :)")
		input := pricetypes.Input{
			Symbols:    c.symbols,
			Multiplier: BandPriceMultiplier,
		}
		calldata := obi.MustEncode(input)
		// TODO: change to some better system obviously
		clientID := string(time.Now().Unix())

		msg := oracletypes.NewMsgRequestData(oracletypes.OracleScriptID(c.oracleScriptID), calldata, c.askCount, c.minCount, clientID, c.requester)
		gasLimit := estimateGas(c, msg)

		hash, err := signAndBroadcast(c, c.keys[0], []sdk.Msg{msg}, gasLimit, "")
		if err != nil {
			// Use info level because this error can happen and retry process can solve this error.
			l.Info(":warning: %s", err.Error())
			time.Sleep(c.rpcPollInterval)
			continue
		}

		_ = hash
		time.Sleep(30 * time.Second)

		_ = ctx
	}
}

func runCmd(c *Context) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "run",
		Aliases: []string{"r"},
		Short:   "Run the price feed requester process",
		Args:    cobra.ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			if cfg.ChainID == "" {
				return errors.New("Chain ID must not be empty")
			}
			keys, err := keybase.List()
			if err != nil {
				return err
			}
			if len(keys) == 0 {
				return errors.New("No key available")
			}
			c.keys = keys
			c.requester, err = sdk.AccAddressFromBech32(cfg.Requester)
			if err != nil {
				return err
			}

			c.oracleScriptID = cfg.OracleScriptID
			c.askCount = cfg.AskCount
			c.minCount = cfg.MinCount

			c.gasPrices, err = sdk.ParseDecCoins(cfg.GasPrices)
			if err != nil {
				return err
			}
			allowLevel, err := log.AllowLevel(cfg.LogLevel)
			if err != nil {
				return err
			}
			l := NewLogger(allowLevel)
			l.Info(":star: Creating HTTP client with node URI: %s", cfg.NodeURI)
			c.client, err = httpclient.New(cfg.NodeURI, "/websocket")
			if err != nil {
				return err
			}
			c.fileCache = filecache.New(filepath.Join(viper.GetString(flags.FlagHome), "files"))
			c.broadcastTimeout, err = time.ParseDuration(cfg.BroadcastTimeout)
			if err != nil {
				return err
			}
			c.maxTry = cfg.MaxTry
			c.rpcPollInterval, err = time.ParseDuration(cfg.RPCPollInterval)
			if err != nil {
				return err
			}
			c.metricsEnabled = cfg.MetricsListenAddr != ""
			return runImpl(c, l)
		},
	}
	cmd.Flags().String(flags.FlagChainID, "", "chain ID of BandChain network")
	cmd.Flags().String(flags.FlagNode, "tcp://localhost:26657", "RPC url to BandChain node")
	cmd.Flags().String(flagRequester, "", "validator address")
	cmd.Flags().Int64(flagOracleScriptID, 37, "oracle scriptID")
	cmd.Flags().Uint64(flagAskCount, 3, "ask count")
	cmd.Flags().Uint64(flagMinCount, 5, "min count")
	cmd.Flags().StringSlice(flagSymbols, []string{"BTC", "ETH"}, "symbols")
	cmd.Flags().String(flags.FlagGasPrices, "", "gas prices for report transaction")
	cmd.Flags().String(flagLogLevel, "info", "set the logger level")
	cmd.Flags().String(flagBroadcastTimeout, "5m", "The time that vader will wait for tx commit")
	cmd.Flags().String(flagRPCPollInterval, "1s", "The duration of rpc poll interval")
	cmd.Flags().Uint64(flagMaxTry, 5, "The maximum number of tries to submit a report transaction")
	viper.BindPFlag(flags.FlagChainID, cmd.Flags().Lookup(flags.FlagChainID))
	viper.BindPFlag(flags.FlagNode, cmd.Flags().Lookup(flags.FlagNode))
	viper.BindPFlag(flagRequester, cmd.Flags().Lookup(flagRequester))
	viper.BindPFlag(flags.FlagGasPrices, cmd.Flags().Lookup(flags.FlagGasPrices))
	viper.BindPFlag(flagLogLevel, cmd.Flags().Lookup(flagLogLevel))
	viper.BindPFlag(flagBroadcastTimeout, cmd.Flags().Lookup(flagBroadcastTimeout))
	viper.BindPFlag(flagRPCPollInterval, cmd.Flags().Lookup(flagRPCPollInterval))
	viper.BindPFlag(flagMaxTry, cmd.Flags().Lookup(flagMaxTry))
	return cmd
}
