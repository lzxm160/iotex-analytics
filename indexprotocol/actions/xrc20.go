// Copyright (c) 2019 IoTeX
// This is an alpha (internal) release and is not suitable for production. This source code is provided 'as is' and no
// warranties are given as to title or non-infringement, merchantability or fitness for purpose and, to the extent
// permitted by law, all liability for your use of the code is disclaimed. This source code is governed by Apache
// License 2.0 that can be found in the LICENSE file.

package actions

import (
	"context"
	"database/sql"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/iotexproject/go-pkgs/hash"
	"github.com/iotexproject/iotex-core/blockchain/block"
)

const (
	// Xrc20HistoryTableName is the table name of xrc20 history
	Xrc20HistoryTableName = "xrc20_history"
	// transferSha3 is sha3 of xrc20's transfer
	transferSha3 = "a9059cbb"
)

type (
	// Xrc20History defines the base schema of "xrc20 history" table
	Xrc20History struct {
		ActionHash  string
		ReceiptHash string
		Address     string
		Data        []byte
		BlockHeight uint64
		Index       uint64
		Timestamp   uint64
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
	if _, err := p.Store.GetDB().Exec(fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (action_hash VARCHAR(64) NOT NULL, receipt_hash VARCHAR(64) NOT NULL UNIQUE, address VARCHAR(41) NOT NULL,`data` TEXT,block_height DECIMAL(65, 0), `index` DECIMAL(65, 0),`timestamp` DECIMAL(65, 0),status VARCHAR(7) NOT NULL), PRIMARY KEY (action_hash,receipt_hash)",
		Xrc20HistoryTableName)); err != nil {
		return err
	}
	return nil
}

// updateXrc20History stores Xrc20 information into Xrc20 history table
func (p *Protocol) updateXrc20History(
	tx *sql.Tx,
	blk *block.Block,
) error {
	valStrs := make([]string, 0)
	valArgs := make([]interface{}, 0)
	for _, receipt := range blk.Receipts {
		receiptStatus := "failure"
		if receipt.Status == uint64(1) {
			receiptStatus = "success"
		}
		for _, l := range receipt.Logs {
			data := string(l.Data)
			//if !strings.EqualFold(data[:8], transferSha3) {
			//	continue
			//}
			fmt.Println(data)
			fmt.Println(hex.EncodeToString(l.Data))
			fmt.Println(l.Index)
			ah := hex.EncodeToString(l.ActionHash[:])
			receiptHash := receipt.Hash()

			rh := hex.EncodeToString(receiptHash[:])
			valStrs = append(valStrs, "(?, ?, ?, ?, ?, ?, ?, ?)")
			valArgs = append(valArgs, ah, rh, l.Address, data, l.BlockHeight, l.Index, blk.Timestamp().Unix(), receiptStatus)
		}
	}
	if len(valArgs) == 0 {
		return nil
	}
	insertQuery := fmt.Sprintf("INSERT IGNORE INTO %s (action_hash, receipt_hash, address,`data`,block_height, `index`,`timestamp`,status) VALUES %s", Xrc20HistoryTableName, strings.Join(valStrs, ","))

	if _, err := tx.Exec(insertQuery, valArgs...); err != nil {
		return err
	}
	return nil
}
