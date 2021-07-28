package vader

import (
	"sync/atomic"
	"time"

	"github.com/cosmos/cosmos-sdk/crypto/keys"
	sdk "github.com/cosmos/cosmos-sdk/types"
	rpcclient "github.com/tendermint/tendermint/rpc/client"

	"github.com/bandprotocol/bandchain/chain/pkg/filecache"
)

type Context struct {
	client           rpcclient.Client
	requester        sdk.AccAddress
	oracleScriptID   int64
	askCount         uint64
	minCount         uint64
	symbols          []string
	gasPrices        sdk.DecCoins
	keys             []keys.Info
	fileCache        filecache.Cache
	broadcastTimeout time.Duration
	maxTry           uint64
	rpcPollInterval  time.Duration

	metricsEnabled bool
	handlingGauge  int64
	pendingGauge   int64
	errorCount     int64
	submittedCount int64
}

func (c *Context) updateErrorCount(amount int64) {
	if c.metricsEnabled {
		atomic.AddInt64(&c.errorCount, amount)
	}
}

func (c *Context) updateSubmittedCount(amount int64) {
	if c.metricsEnabled {
		atomic.AddInt64(&c.submittedCount, amount)
	}
}
