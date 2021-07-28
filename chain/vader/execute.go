package vader

import (
	"fmt"

	sdkCtx "github.com/cosmos/cosmos-sdk/client/context"
	ckeys "github.com/cosmos/cosmos-sdk/client/keys"
	"github.com/cosmos/cosmos-sdk/crypto/keys"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth"

	"github.com/bandprotocol/bandchain/chain/app"
)

var (
	cdc = app.MakeCodec()
)

func signAndBroadcast(
	c *Context, key keys.Info, msgs []sdk.Msg, gasLimit uint64, memo string,
) (string, error) {
	cliCtx := sdkCtx.CLIContext{Client: c.client, TrustNode: true, Codec: cdc}
	acc, err := auth.NewAccountRetriever(cliCtx).GetAccount(key.GetAddress())
	if err != nil {
		return "", fmt.Errorf("Failed to retreive account with error: %s", err.Error())
	}

	txBldr := auth.NewTxBuilder(
		auth.DefaultTxEncoder(cdc), acc.GetAccountNumber(), acc.GetSequence(),
		gasLimit, 1, false, cfg.ChainID, memo, sdk.NewCoins(), c.gasPrices,
	)
	// txBldr, err = authclient.EnrichWithGas(txBldr, cliCtx, []sdk.Msg{msg})
	// if err != nil {
	// 	l.Error(":exploding_head: Failed to enrich with gas with error: %s", c, err.Error())
	// 	return
	// }

	out, err := txBldr.WithKeybase(keybase).BuildAndSign(key.GetName(), ckeys.DefaultKeyPass, msgs)
	if err != nil {
		return "", fmt.Errorf("Failed to build tx with error: %s", err.Error())
	}

	res, err := cliCtx.BroadcastTxCommit(out)
	if err != nil {
		return "", fmt.Errorf("Failed to broadcast tx with error: %s", err.Error())
	}
	return res.TxHash, nil
}

