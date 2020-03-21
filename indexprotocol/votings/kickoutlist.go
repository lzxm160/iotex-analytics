// Copyright (c) 2020 IoTeX
// This is an alpha (internal) release and is not suitable for production. This source code is provided 'as is' and no
// warranties are given as to title or non-infringement, merchantability or fitness for purpose and, to the extent
// permitted by law, all liability for your use of the code is disclaimed. This source code is governed by Apache
// License 2.0 that can be found in the LICENSE file.

package votings

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/golang/protobuf/proto"
	"github.com/pkg/errors"

	"github.com/iotexproject/go-pkgs/byteutil"
	"github.com/iotexproject/iotex-proto/golang/iotexapi"
	"github.com/iotexproject/iotex-proto/golang/iotextypes"
)

const (
	// KickoutListTableName is the table name of kickout list
	KickoutListTableName = "kickout_list"
	// EpochAddressIndexName is the index name of epoch number and address on kickout table
	EpochAddressIndexName = "epoch_address_index"
	createKickoutList     = "CREATE TABLE IF NOT EXISTS %s " +
		"(epoch_number DECIMAL(65, 0) NOT NULL,intensity_rate DECIMAL(65, 0) NOT NULL,address VARCHAR(41) NOT NULL, count DECIMAL(65, 0) NOT NULL,PRIMARY KEY (`epoch_number`, `address`), UNIQUE KEY %s (epoch_number, address))"
	insertKickoutList = "INSERT IGNORE INTO %s (epoch_number,intensity_rate,address,count) VALUES (?, ?, ?, ?)"
)

type (
	// KickoutList defines the schema of "kickout_list" table
	KickoutList struct {
		EpochNumber   uint64
		IntensityRate uint64
		Address       string
		Count         uint64
	}
)

func (p *Protocol) createKickoutListTable(tx *sql.Tx) error {
	if _, err := tx.Exec(fmt.Sprintf(createKickoutList, KickoutListTableName, EpochAddressIndexName)); err != nil {
		return err
	}
	return nil
}

func (p *Protocol) updateKickoutListTable(cli iotexapi.APIServiceClient, epochNum uint64, tx *sql.Tx) error {
	kickoutList, err := p.getKickoutList(cli, epochNum)
	if err != nil {
		return err
	}
	insertQuery := fmt.Sprintf(insertKickoutList, KickoutListTableName)
	for _, k := range kickoutList.Blacklists {
		if _, err := tx.Exec(insertQuery, epochNum, kickoutList.IntensityRate, k.Address, k.Count); err != nil {
			return errors.Wrap(err, "failed to update kickout list table")
		}
	}
	return nil
}

func (p *Protocol) getKickoutList(cli iotexapi.APIServiceClient, epochNum uint64) (*iotextypes.KickoutCandidateList, error) {
	request := &iotexapi.ReadStateRequest{
		ProtocolID: []byte("poll"),
		MethodName: []byte("KickoutListByEpoch"),
		Arguments:  [][]byte{byteutil.Uint64ToBytes(epochNum)},
	}
	out, err := cli.ReadState(context.Background(), request)
	if err != nil {
		fmt.Println("getKickoutList:", err)
		return nil, nil
		//return nil, err
	}
	pb := &iotextypes.KickoutCandidateList{}
	if err := proto.Unmarshal(out.Data, pb); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal candidate")
	}
	return pb, nil
}
