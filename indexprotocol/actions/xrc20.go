// Copyright (c) 2019 IoTeX
// This is an alpha (internal) release and is not suitable for production. This source code is provided 'as is' and no
// warranties are given as to title or non-infringement, merchantability or fitness for purpose and, to the extent
// permitted by law, all liability for your use of the code is disclaimed. This source code is governed by Apache
// License 2.0 that can be found in the LICENSE file.

package actions

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/iotexproject/go-pkgs/hash"
	"github.com/iotexproject/iotex-core/blockchain/block"

	"github.com/iotexproject/iotex-analytics/indexprotocol/blocks"
)

const (
	// Xrc20HistoryTableName is the table name of xrc20 history
	Xrc20HistoryTableName = "xrc20_history"
)

type (
	// Xrc20History defines the base schema of "xrc20 history" table
	Xrc20History struct {
		ActionType    string
		ActionHash    string
		ReceiptHash   string
		BlockHeight   uint64
		From          string
		To            string
		GasPrice      string
		GasConsumed   uint64
		Nonce         uint64
		Amount        string
		ReceiptStatus string
	}

	// Xrc20Info defines an Xrc20's information
	Xrc20Info struct {
		ReceiptHash hash.Hash256
		From        string
		To          string
		GasPrice    string
		Nonce       uint64
		Amount      string
		Data        string
		timestamp   time.Time
	}
)

// CreateTables creates tables
func (p *Protocol) CreateXrc20Tables(ctx context.Context) error {
	// create block by action table
	if _, err := p.Store.GetDB().Exec(fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s "+
		"(action_type TEXT NOT NULL, action_hash VARCHAR(64) NOT NULL, receipt_hash VARCHAR(64) NOT NULL UNIQUE, block_height DECIMAL(65, 0) NOT NULL, "+
		"`from` VARCHAR(41) NOT NULL, `to` VARCHAR(41) NOT NULL, gas_price DECIMAL(65, 0) NOT NULL, gas_consumed DECIMAL(65, 0) NOT NULL, nonce DECIMAL(65, 0) NOT NULL, "+
		"amount DECIMAL(65, 0) NOT NULL, receipt_status TEXT NOT NULL, data TEXT, timestamp DECIMAL(65, 0) NOT NULL, PRIMARY KEY (action_hash), FOREIGN KEY (block_height) REFERENCES %s(block_height))",
		Xrc20HistoryTableName, blocks.BlockHistoryTableName)); err != nil {
		return err
	}
	return nil
}

// HandleXrc20 handles Xrc20
func (p *Protocol) HandleXrc20(ctx context.Context, tx *sql.Tx, blk *block.Block) error {
	return nil
}
