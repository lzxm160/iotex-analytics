// Copyright (c) 2020 IoTeX
// This is an alpha (internal) release and is not suitable for production. This source code is provided 'as is' and no
// warranties are given as to title or non-infringement, merchantability or fitness for purpose and, to the extent
// permitted by law, all liability for your use of the code is disclaimed. This source code is governed by Apache
// License 2.0 that can be found in the LICENSE file.

package votings

import (
	"context"
	"fmt"

	"github.com/golang/protobuf/proto"
	"github.com/pkg/errors"

	"github.com/iotexproject/iotex-core/pkg/util/byteutil"
	"github.com/iotexproject/iotex-proto/golang/iotexapi"
	"github.com/iotexproject/iotex-proto/golang/iotextypes"

	"github.com/iotexproject/iotex-analytics/indexcontext"
)

const (
	// KickoutListTableName is the table name of kickout list
	KickoutListTableName = "kickout_list"

	createKickoutList = "CREATE TABLE IF NOT EXISTS %s " +
		"(epoch_number DECIMAL(65, 0) NOT NULL, delegate_name VARCHAR(255) NOT NULL, operator_address VARCHAR(41) NOT NULL, " +
		"reward_address VARCHAR(41) NOT NULL, total_weighted_votes DECIMAL(65, 0) NOT NULL, self_staking DECIMAL(65,0) NOT NULL, " +
		"block_reward_percentage INT DEFAULT 100, epoch_reward_percentage INT DEFAULT 100, foundation_bonus_percentage INT DEFAULT 100, " +
		"staking_address VARCHAR(40) DEFAULT %s)"
)

type (
// VotingResult defines the schema of "voting result" table
//VotingResult struct {
//	EpochNumber               uint64
//	DelegateName              string
//	OperatorAddress           string
//	RewardAddress             string
//	TotalWeightedVotes        string
//	SelfStaking               string
//	BlockRewardPercentage     uint64
//	EpochRewardPercentage     uint64
//	FoundationBonusPercentage uint64
//	StakingAddress            string
//}
)

// CreateTables creates tables
func (p *Protocol) createKickoutListTable(ctx context.Context) error {
	return nil
}

// HandleBlock handles blocks
func (p *Protocol) updateKickoutListTable(ctx context.Context) error {
	indexCtx := indexcontext.MustGetIndexCtx(ctx)
	if indexCtx.ChainClient == nil {
		return errors.New("chain client error")
	}
	for i := uint64(100); i < 5000; i += 100 {
		request := &iotexapi.ReadStateRequest{
			ProtocolID: []byte("poll"),
			MethodName: []byte("KickoutListByEpoch"),
			Arguments:  [][]byte{byteutil.Uint64ToBytes(i)},
		}
		out, err := indexCtx.ChainClient.ReadState(context.Background(), request)
		if err != nil {
			return err
		}
		pb := &iotextypes.KickoutCandidateList{}
		if err := proto.Unmarshal(out.Data, pb); err != nil {
			return errors.Wrap(err, "failed to unmarshal candidate")
		}
		fmt.Println("pb:", pb.String())
	}

	return nil
}
