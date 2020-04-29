// Copyright (c) 2020 IoTeX
// This is an alpha (internal) release and is not suitable for production. This source code is provided 'as is' and no
// warranties are given as to title or non-infringement, merchantability or fitness for purpose and, to the extent
// permitted by law, all liability for your use of the code is disclaimed. This source code is governed by Apache
// License 2.0 that can be found in the LICENSE file.

package votings

import (
	"context"
	"fmt"
	"strconv"
	"testing"

	"github.com/gogo/protobuf/proto"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"

	"github.com/golang/protobuf/ptypes/timestamp"
	"github.com/iotexproject/iotex-core/action/protocol/poll"
	"github.com/iotexproject/iotex-core/test/mock/mock_apiserviceclient"
	"github.com/iotexproject/iotex-proto/golang/iotexapi"
	"github.com/iotexproject/iotex-proto/golang/iotextypes"

	"github.com/iotexproject/iotex-analytics/epochctx"
	"github.com/iotexproject/iotex-analytics/indexprotocol"
	s "github.com/iotexproject/iotex-analytics/sql"
)

const (
	localconnectStr = "root:123456@tcp(192.168.146.140:3306)/"
	localdbName     = "analytics"
)

func TestXX(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	chainClient := mock_apiserviceclient.NewMockServiceClient(ctrl)
	mock(chainClient, t)
	height := uint64(110000)
	epochNumber := uint64(68888)
	require := require.New(t)
	store := s.NewMySQL(localconnectStr, localdbName)
	require.NoError(store.Start(context.Background()))
	cfg := indexprotocol.VoteWeightCalConsts{
		DurationLg: 1.2,
		AutoStake:  1,
		SelfStake:  1.05,
	}
	p, err := NewProtocol(store, epochctx.NewEpochCtx(36, 24, 15), indexprotocol.GravityChain{}, indexprotocol.Poll{
		VoteThreshold:        "100000000000000000000",
		ScoreThreshold:       "0",
		SelfStakingThreshold: "0",
	}, cfg)
	require.NoError(err)
	require.NoError(p.stakingV2(chainClient, height, epochNumber, nil))

	// checkout bucket if it's written right
	tx, err := p.Store.GetDB().Begin()
	require.NoError(err)
	ret, err := p.nativeV2BucketTableOperator.Get(height, p.Store.GetDB(), tx)
	require.NoError(err)
	buckets, ok := ret.(*iotextypes.VoteBucketList)
	require.True(ok)
	fmt.Println(buckets.Buckets[0])

	// checkout candidate if it's written right
	fmt.Println("//////////////////////////////")
	ret, err = p.nativeV2CandidateTableOperator.Get(height, p.Store.GetDB(), tx)
	require.NoError(err)
	candidates, ok := ret.(*iotextypes.CandidateListV2)
	require.True(ok)
	fmt.Println(candidates.Candidates[0])
}

func mock(chainClient *mock_apiserviceclient.MockServiceClient, t *testing.T) {
	require := require.New(t)
	readStateRequest := &iotexapi.ReadStateRequest{
		ProtocolID: []byte(poll.ProtocolID),
		MethodName: []byte(strconv.FormatInt(int64(iotexapi.ReadStakingDataMethod_BUCKETS), 10)),
		Arguments:  [][]byte{[]byte(strconv.FormatUint(0, 10)), []byte(strconv.FormatUint(100, 10))},
	}
	buckets := []*iotextypes.VoteBucket{
		&iotextypes.VoteBucket{
			Index:            10,
			CandidateAddress: "io1mflp9m6hcgm2qcghchsdqj3z3eccrnekx9p0ms",
			StakedAmount:     "30000",
			StakedDuration:   20,
			CreateTime:       &timestamp.Timestamp{Seconds: int64(1587864599), Nanos: int32(456)},
			StakeStartTime:   &timestamp.Timestamp{Seconds: int64(1587864599), Nanos: int32(123)},
			UnstakeStartTime: &timestamp.Timestamp{Seconds: int64(1587864599), Nanos: int32(321)},
			AutoStake:        false,
			Owner:            "io1mflp9m6hcgm2qcghchsdqj3z3eccrnekx9p0ms",
		},
	}
	vbl := &iotextypes.VoteBucketList{Buckets: buckets}
	s, err := proto.Marshal(vbl)
	require.NoError(err)
	first := chainClient.EXPECT().ReadState(gomock.Any(), readStateRequest).Times(1).Return(&iotexapi.ReadStateResponse{
		Data: s,
	}, nil)

	// mock candidate
	readStateRequest = &iotexapi.ReadStateRequest{
		ProtocolID: []byte(poll.ProtocolID),
		MethodName: []byte(strconv.FormatInt(int64(iotexapi.ReadStakingDataMethod_CANDIDATES), 10)),
		Arguments:  [][]byte{[]byte(strconv.FormatUint(0, 10)), []byte(strconv.FormatUint(100, 10))},
	}
	candidates := []*iotextypes.CandidateV2{
		&iotextypes.CandidateV2{
			OwnerAddress:       "io1mflp9m6hcgm2qcghchsdqj3z3eccrnekx9p0ms",
			OperatorAddress:    "io1mflp9m6hcgm2qcghchsdqj3z3eccrnekx9p0ms",
			RewardAddress:      "io1mflp9m6hcgm2qcghchsdqj3z3eccrnekx9p0ms",
			Name:               "io1mflp9m6hcgm2qcghchsdqj3z3eccrnekx9p0ms",
			TotalWeightedVotes: "888886",
			SelfStakeBucketIdx: 6666,
			SelfStakingTokens:  "99999",
		},
	}
	cl := &iotextypes.CandidateListV2{Candidates: candidates}
	s, err = proto.Marshal(cl)
	require.NoError(err)
	second := chainClient.EXPECT().ReadState(gomock.Any(), readStateRequest).Times(1).Return(&iotexapi.ReadStateResponse{
		Data: s,
	}, nil)

	gomock.InOrder(
		first,
		second,
	)
}
