package vader

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/bandprotocol/bandchain/chain/x/oracle/types"
)

// Constant used to estimate gas price of reports transaction.
const (
	// cosmos
	baseFixedGas        = uint64(37764)
	baseTransactionSize = uint64(200)
	txCostPerByte       = uint64(5) // Using DefaultTxSizeCostPerByte of BandChain

	payingFeeCost = uint64(16500)
)

func estimateTxSize(msgs []sdk.Msg) uint64 {
	// base tx + reports
	size := baseTransactionSize

	for _, msg := range msgs {
		msg, ok := msg.(types.MsgReportData)
		if !ok {
			panic("Don't support non-report data message")
		}

		ser := cdc.MustMarshalBinaryBare(msg)
		size += uint64(len(ser))
	}

	return size
}

func estimateGas(c *Context, msgs ...sdk.Msg) uint64 {
	gas := baseFixedGas

	txSize := estimateTxSize(msgs)
	gas += txCostPerByte * txSize

	// process paying fee
	if len(c.gasPrices) > 0 {
		gas += payingFeeCost
	}

	return gas
}
