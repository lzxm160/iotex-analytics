// Copyright (c) 2019 IoTeX
// This is an alpha (internal) release and is not suitable for production. This source code is provided 'as is' and no
// warranties are given as to title or non-infringement, merchantability or fitness for purpose and, to the extent
// permitted by law, all liability for your use of the code is disclaimed. This source code is governed by Apache
// License 2.0 that can be found in the LICENSE file.

package rewards

import (
	"context"
	"database/sql"
	"encoding/hex"
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/iotexproject/iotex-proto/golang/iotextypes"
	"github.com/pkg/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"google.golang.org/grpc"

	"github.com/golang/mock/gomock"
	"github.com/golang/protobuf/proto"
	"github.com/stretchr/testify/require"

	"github.com/iotexproject/iotex-core/test/mock/mock_apiserviceclient"
	"github.com/iotexproject/iotex-election/pb/api"
	mock_election "github.com/iotexproject/iotex-election/test/mock/mock_apiserviceclient"
	"github.com/iotexproject/iotex-proto/golang/iotexapi"

	"github.com/iotexproject/iotex-analytics/epochctx"
	"github.com/iotexproject/iotex-analytics/indexcontext"
	"github.com/iotexproject/iotex-analytics/indexprotocol"
	s "github.com/iotexproject/iotex-analytics/sql"
	"github.com/iotexproject/iotex-analytics/testutil"
)

const (
	connectStr = "ba8df54bd3754e:9cd1f263@tcp(us-cdbr-iron-east-02.cleardb.net:3306)/"
	dbName     = "heroku_7fed0b046078f80"
)

func TestProtocol(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	require := require.New(t)
	ctx := context.Background()

	testutil.CleanupDatabase(t, connectStr, dbName)

	store := s.NewMySQL(connectStr, dbName)
	require.NoError(store.Start(ctx))
	defer func() {
		_, err := store.GetDB().Exec("DROP DATABASE " + dbName)
		require.NoError(err)
		require.NoError(store.Stop(ctx))
	}()

	p := NewProtocol(store, epochctx.NewEpochCtx(1, 1, 1), indexprotocol.Rewarding{})

	require.NoError(p.CreateTables(ctx))

	blk1, err := testutil.BuildCompleteBlock(uint64(1), uint64(2))
	require.NoError(err)

	chainClient := mock_apiserviceclient.NewMockServiceClient(ctrl)
	electionClient := mock_election.NewMockAPIServiceClient(ctrl)
	ctx = indexcontext.WithIndexCtx(context.Background(), indexcontext.IndexCtx{
		ChainClient:     chainClient,
		ElectionClient:  electionClient,
		ConsensusScheme: "ROLLDPOS",
	})

	chainClient.EXPECT().ReadState(gomock.Any(), gomock.Any()).Times(1).Return(&iotexapi.ReadStateResponse{
		Data: []byte(strconv.FormatUint(1000, 10)),
	}, nil)
	electionClient.EXPECT().GetCandidates(gomock.Any(), gomock.Any()).Times(1).Return(
		&api.CandidateResponse{
			Candidates: []*api.Candidate{
				{
					Name:          "616c6661",
					RewardAddress: testutil.RewardAddr1,
				},
				{
					Name:          "627261766f",
					RewardAddress: testutil.RewardAddr2,
				},
				{
					Name:          "636861726c6965",
					RewardAddress: testutil.RewardAddr3,
				},
			},
		}, nil,
	)

	require.NoError(store.Transact(func(tx *sql.Tx) error {
		return p.HandleBlock(ctx, tx, blk1)
	}))

	actionHash1 := blk1.Actions[4].Hash()
	rewardHistoryList, err := p.getRewardHistory(hex.EncodeToString(actionHash1[:]))
	require.NoError(err)
	require.Equal(1, len(rewardHistoryList))
	require.Equal(uint64(1), rewardHistoryList[0].EpochNumber)
	require.Equal("616c6661", rewardHistoryList[0].CandidateName)
	require.Equal(testutil.RewardAddr1, rewardHistoryList[0].RewardAddress)
	require.Equal("16", rewardHistoryList[0].BlockReward)
	require.Equal("0", rewardHistoryList[0].EpochReward)
	require.Equal("0", rewardHistoryList[0].FoundationBonus)

	actionHash2 := blk1.Actions[5].Hash()
	rewardHistoryList, err = p.getRewardHistory(hex.EncodeToString(actionHash2[:]))
	require.NoError(err)
	require.Equal(3, len(rewardHistoryList))
}

func TestUpdateCandidateRewardAddress(t *testing.T) {
	// TODO
	localconnectStr := "root:123456@tcp(192.168.146.140:3306)/"
	localdbName := "analytics"
	//chainEndpoint := "api.iotex.one:80"
	chainEndpoint := "api.testnet.iotex.one:80"
	require := require.New(t)

	ctx := context.Background()

	store := s.NewMySQL(localconnectStr, localdbName)
	require.NoError(store.Start(ctx))
	defer func() {
		require.NoError(store.Stop(ctx))
	}()
	epochctx := epochctx.NewEpochCtx(
		36,
		24,
		15,
		epochctx.EnableDardanellesSubEpoch(1816201, 30),
		epochctx.FairbankHeight(3252241),
	)
	p := NewProtocol(store, epochctx, indexprotocol.Rewarding{})

	require.NoError(p.CreateTables(ctx))
	grpcCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	conn, err := grpc.DialContext(grpcCtx, chainEndpoint, grpc.WithBlock(), grpc.WithInsecure())
	require.NoError(err)
	chainClient := iotexapi.NewAPIServiceClient(conn)
	require.NoError(p.updateCandidateRewardAddress(chainClient, nil, 3253241))

	fmt.Println("--------------------------")
	cl, err := getCandidatesV2(chainClient, 0, 100)
	require.NoError(err)
	fmt.Println("len(cl.Candidates):", len(cl.Candidates))
	for _, c := range cl.Candidates {
		fmt.Println(c)
	}

	fmt.Println("--------------------------")
	buckets, err := getBucketsV2(chainClient, 0, 100)
	require.NoError(err)
	fmt.Println("len(buckets.Buckets):", len(buckets.Buckets))
	for _, b := range buckets.Buckets {
		fmt.Println(b)
	}
}

func getCandidatesV2(chainClient iotexapi.APIServiceClient, offset, limit uint32) (candidateList *iotextypes.CandidateListV2, err error) {
	methodName, err := proto.Marshal(&iotexapi.ReadStakingDataMethod{
		Method: iotexapi.ReadStakingDataMethod_CANDIDATES,
	})
	if err != nil {
		return nil, err
	}
	arg, err := proto.Marshal(&iotexapi.ReadStakingDataRequest{
		Request: &iotexapi.ReadStakingDataRequest_Candidates_{
			Candidates: &iotexapi.ReadStakingDataRequest_Candidates{
				Pagination: &iotexapi.PaginationParam{
					Offset: offset,
					Limit:  limit,
				},
			},
		},
	})
	if err != nil {
		return nil, err
	}
	readStateRequest := &iotexapi.ReadStateRequest{
		ProtocolID: []byte("staking"),
		MethodName: methodName,
		Arguments:  [][]byte{arg},
	}
	readStateRes, err := chainClient.ReadState(context.Background(), readStateRequest)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			// TODO rm this when commit pr
			fmt.Println("ReadStakingDataMethod_BUCKETS not found")
		}
		return
	}
	candidateList = &iotextypes.CandidateListV2{}
	if err := proto.Unmarshal(readStateRes.GetData(), candidateList); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal VoteBucketList")
	}
	return
}

func getBucketsV2(chainClient iotexapi.APIServiceClient, offset, limit uint32) (voteBucketList *iotextypes.VoteBucketList, err error) {
	methodName, err := proto.Marshal(&iotexapi.ReadStakingDataMethod{
		Method: iotexapi.ReadStakingDataMethod_BUCKETS,
	})
	if err != nil {
		return nil, err
	}
	arg, err := proto.Marshal(&iotexapi.ReadStakingDataRequest{
		Request: &iotexapi.ReadStakingDataRequest_Buckets{
			Buckets: &iotexapi.ReadStakingDataRequest_VoteBuckets{
				Pagination: &iotexapi.PaginationParam{
					Offset: offset,
					Limit:  limit,
				},
			},
		},
	})
	if err != nil {
		return nil, err
	}
	readStateRequest := &iotexapi.ReadStateRequest{
		ProtocolID: []byte("staking"),
		MethodName: methodName,
		Arguments:  [][]byte{arg},
	}
	readStateRes, err := chainClient.ReadState(context.Background(), readStateRequest)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			// TODO rm this when commit pr
			fmt.Println("ReadStakingDataMethod_BUCKETS not found")
		}
		return
	}
	voteBucketList = &iotextypes.VoteBucketList{}
	if err := proto.Unmarshal(readStateRes.GetData(), voteBucketList); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal VoteBucketList")
	}
	return
}
