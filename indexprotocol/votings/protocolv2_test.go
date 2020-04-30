// Copyright (c) 2020 IoTeX
// This is an alpha (internal) release and is not suitable for production. This source code is provided 'as is' and no
// warranties are given as to title or non-infringement, merchantability or fitness for purpose and, to the extent
// permitted by law, all liability for your use of the code is disclaimed. This source code is governed by Apache
// License 2.0 that can be found in the LICENSE file.

package votings

import (
	"context"
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/golang/mock/gomock"
	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/timestamp"
	"github.com/stretchr/testify/require"

	"github.com/iotexproject/iotex-core/test/mock/mock_apiserviceclient"
	"github.com/iotexproject/iotex-proto/golang/iotexapi"
	"github.com/iotexproject/iotex-proto/golang/iotextypes"

	"github.com/iotexproject/iotex-analytics/epochctx"
	"github.com/iotexproject/iotex-analytics/indexprotocol"
	s "github.com/iotexproject/iotex-analytics/sql"
)

const (
	localconnectStr = "root:123456@tcp(192.168.146.140:3306)/"
	//localconnectStr = connectStr
	localdbName = "analytics"
	//localdbName = dbName
)

var (
	buckets = []*iotextypes.VoteBucket{
		&iotextypes.VoteBucket{
			Index:            10,
			CandidateAddress: "io1mflp9m6hcgm2qcghchsdqj3z3eccrnekx9p0ms",
			StakedAmount:     "30000",
			StakedDuration:   86400, // one day
			CreateTime:       &timestamp.Timestamp{Seconds: int64(1587864599), Nanos: 0},
			StakeStartTime:   &timestamp.Timestamp{Seconds: int64(1587864599), Nanos: 0},
			UnstakeStartTime: &timestamp.Timestamp{Seconds: int64(1587864599), Nanos: 0},
			AutoStake:        false,
			Owner:            "io1mflp9m6hcgm2qcghchsdqj3z3eccrnekx9p0ms",
		},
		&iotextypes.VoteBucket{
			Index:            11,
			CandidateAddress: "io1mflp9m6hcgm2qcghchsdqj3z3eccrnekx9p0ms",
			StakedAmount:     "30000",
			StakedDuration:   86400,
			CreateTime:       &timestamp.Timestamp{Seconds: int64(1587864599), Nanos: 0},
			StakeStartTime:   &timestamp.Timestamp{Seconds: int64(1587864599), Nanos: 0},
			UnstakeStartTime: &timestamp.Timestamp{Seconds: int64(1587864599), Nanos: 0},
			AutoStake:        false,
			Owner:            "io1ph0u2psnd7muq5xv9623rmxdsxc4uapxhzpg02",
		},
		&iotextypes.VoteBucket{
			Index:            12,
			CandidateAddress: "io1mflp9m6hcgm2qcghchsdqj3z3eccrnekx9p0ms",
			StakedAmount:     "30000",
			StakedDuration:   86400,
			CreateTime:       &timestamp.Timestamp{Seconds: int64(1587864599), Nanos: int32(456)},
			StakeStartTime:   &timestamp.Timestamp{Seconds: int64(1587864599), Nanos: int32(123)},
			UnstakeStartTime: &timestamp.Timestamp{Seconds: int64(1587864599), Nanos: int32(321)},
			AutoStake:        false,
			Owner:            "io1vdtfpzkwpyngzvx7u2mauepnzja7kd5rryp0sg",
		},
	}
	candidates = []*iotextypes.CandidateV2{
		&iotextypes.CandidateV2{
			OwnerAddress:       "io1mflp9m6hcgm2qcghchsdqj3z3eccrnekx9p0ms",
			OperatorAddress:    "io1mflp9m6hcgm2qcghchsdqj3z3eccrnekx9p0ms",
			RewardAddress:      "io1mflp9m6hcgm2qcghchsdqj3z3eccrnekx9p0ms",
			Name:               "xxxx",
			TotalWeightedVotes: "888886",
			SelfStakeBucketIdx: 6666,
			SelfStakingTokens:  "99999",
		},
	}
)

func TestStakingV2(t *testing.T) {
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
	require.NoError(p.CreateTables(context.Background()))
	require.NoError(p.stakingV2(chainClient, height, epochNumber, nil))

	// checkout bucket if it's written right
	require.NoError(err)
	ret, err := p.nativeV2BucketTableOperator.Get(height, p.Store.GetDB(), nil)
	require.NoError(err)
	bucketList, ok := ret.(*iotextypes.VoteBucketList)
	require.True(ok)
	fmt.Println(buckets)
	fmt.Println(bucketList.Buckets)
	require.True(reflect.DeepEqual(buckets, bucketList.Buckets))
	// checkout candidate if it's written right
	fmt.Println("//////////////////////////////")
	ret, err = p.nativeV2CandidateTableOperator.Get(height, p.Store.GetDB(), nil)
	require.NoError(err)
	candidateList, ok := ret.(*iotextypes.CandidateListV2)
	require.True(ok)
	fmt.Println(candidates)
	fmt.Println(candidateList.Candidates)
	require.True(reflect.DeepEqual(candidates, candidateList.Candidates))
}

func TestRemainingTime(t *testing.T) {
	require := require.New(t)
	// case I: now is before start time
	bucketTime := time.Now().Add(time.Second * 100)
	timestamp, _ := ptypes.TimestampProto(bucketTime)
	bucket := &iotextypes.VoteBucket{
		StakeStartTime: timestamp,
		StakedDuration: 100,
	}
	remaining := remainingTime(bucket)
	require.Equal(time.Duration(0), remaining)

	// case II: now is between start time and starttime+stakedduration
	bucketTime = time.Unix(time.Now().Unix()-10, 0)
	timestamp, _ = ptypes.TimestampProto(bucketTime)
	bucket = &iotextypes.VoteBucket{
		StakeStartTime: timestamp,
		StakedDuration: 100,
	}
	remaining = remainingTime(bucket)
	require.True(remaining > 0)

	// case III: now is after starttime+stakedduration
	bucketTime = time.Unix(time.Now().Unix()-200, 0)
	timestamp, _ = ptypes.TimestampProto(bucketTime)
	bucket = &iotextypes.VoteBucket{
		StakeStartTime: timestamp,
		StakedDuration: 100,
	}
	remaining = remainingTime(bucket)
	require.Equal(time.Duration(0), remaining)
}

func mock(chainClient *mock_apiserviceclient.MockServiceClient, t *testing.T) {
	require := require.New(t)
	methodName := &iotexapi.ReadStakingDataMethod{
		Method: iotexapi.ReadStakingDataMethod_BUCKETS,
	}
	methodNameBytes, _ := proto.Marshal(methodName)
	arguments := &iotexapi.ReadStakingDataRequest_VoteBuckets{
		Pagination: &iotexapi.PaginationParam{
			Offset: 0,
			Limit:  10,
		},
	}
	argumentsBytes, _ := proto.Marshal(arguments)
	readStateRequest := &iotexapi.ReadStateRequest{
		ProtocolID: []byte(protocolID),
		MethodName: methodNameBytes,
		Arguments:  [][]byte{argumentsBytes},
	}

	vbl := &iotextypes.VoteBucketList{Buckets: buckets}
	s, err := proto.Marshal(vbl)
	require.NoError(err)
	first := chainClient.EXPECT().ReadState(gomock.Any(), readStateRequest).Times(1).Return(&iotexapi.ReadStateResponse{
		Data: s,
	}, nil)

	// mock candidate
	methodName = &iotexapi.ReadStakingDataMethod{
		Method: iotexapi.ReadStakingDataMethod_CANDIDATES,
	}
	methodNameBytes, _ = proto.Marshal(methodName)
	arguments2 := &iotexapi.ReadStakingDataRequest_Candidates{
		Pagination: &iotexapi.PaginationParam{
			Offset: 0,
			Limit:  10,
		},
	}
	argumentsBytes, _ = proto.Marshal(arguments2)
	readStateRequest = &iotexapi.ReadStateRequest{
		ProtocolID: []byte(protocolID),
		MethodName: methodNameBytes,
		Arguments:  [][]byte{argumentsBytes},
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
