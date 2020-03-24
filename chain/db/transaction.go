package db

import (
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

func createTransaction(
	txHash []byte,
	timestamp time.Time,
	gasUse uint64,
	gasLimit uint64,
	gasFee sdk.Coins,
	sender sdk.AccAddress,
	success bool,
	blockHeight int64,
) Transaction {
	return Transaction{
		TxHash:      txHash,
		Timestamp:   timestamp.Unix(),
		GasUse:      gasUse,
		GasLimit:    gasLimit,
		GasFee:      gasFee.String(),
		Sender:      sender.String(),
		Success:     success,
		BlockHeight: blockHeight,
	}
}

func (b *BandDB) AddTransaction(
	txHash []byte,
	timestamp time.Time,
	gasUse uint64,
	gasLimit uint64,
	gasFee sdk.Coins,
	sender sdk.AccAddress,
	success bool,
	blockHeight int64,
) error {
	transaction := createTransaction(
		txHash,
		timestamp,
		gasUse,
		gasLimit,
		gasFee,
		sender,
		success,
		blockHeight,
	)
	err := b.tx.Create(&transaction).Error
	return err
}
