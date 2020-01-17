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
	"math/big"
	"testing"
	"time"

	"github.com/iotexproject/iotex-core/pkg/log"
	"google.golang.org/grpc"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"

	"github.com/iotexproject/iotex-core/action/protocol/poll"
	"github.com/iotexproject/iotex-core/pkg/util/byteutil"
	"github.com/iotexproject/iotex-core/state"
	"github.com/iotexproject/iotex-core/test/mock/mock_apiserviceclient"
	"github.com/iotexproject/iotex-election/pb/api"
	mock_election "github.com/iotexproject/iotex-election/test/mock/mock_apiserviceclient"
	"github.com/iotexproject/iotex-proto/golang/iotexapi"

	"github.com/iotexproject/iotex-analytics/epochctx"
	"github.com/iotexproject/iotex-analytics/indexcontext"
	"github.com/iotexproject/iotex-analytics/indexprotocol/blocks"
	s "github.com/iotexproject/iotex-analytics/sql"
	"github.com/iotexproject/iotex-analytics/testutil"
)

func TestXrc20(t *testing.T) {
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

	bp := blocks.NewProtocol(store, epochctx.NewEpochCtx(36, 24, 15))
	p := NewProtocol(store)

	require.NoError(bp.CreateTables(ctx))
	require.NoError(p.CreateTables(ctx))

	chainClient := mock_apiserviceclient.NewMockServiceClient(ctrl)
	electionClient := mock_election.NewMockAPIServiceClient(ctrl)
	bpctx := indexcontext.WithIndexCtx(context.Background(), indexcontext.IndexCtx{
		ChainClient:     chainClient,
		ElectionClient:  electionClient,
		ConsensusScheme: "ROLLDPOS",
	})

	chainClient.EXPECT().ReadState(gomock.Any(), gomock.Any()).Times(1).Return(&iotexapi.ReadStateResponse{
		Data: byteutil.Uint64ToBytes(uint64(1000)),
	}, nil)
	electionClient.EXPECT().GetCandidates(gomock.Any(), gomock.Any()).Times(1).Return(
		&api.CandidateResponse{
			Candidates: []*api.Candidate{
				{
					Name:            "616c6661",
					OperatorAddress: testutil.Addr1,
				},
				{
					Name:            "627261766f",
					OperatorAddress: testutil.Addr2,
				},
			},
		}, nil,
	)
	readStateRequest := &iotexapi.ReadStateRequest{
		ProtocolID: []byte(poll.ProtocolID),
		MethodName: []byte("ActiveBlockProducersByEpoch"),
		Arguments:  [][]byte{byteutil.Uint64ToBytes(uint64(1))},
	}
	candidateList := state.CandidateList{
		{
			Address:       testutil.Addr1,
			RewardAddress: testutil.RewardAddr1,
			Votes:         big.NewInt(100),
		},
		{
			Address:       testutil.Addr2,
			RewardAddress: testutil.RewardAddr2,
			Votes:         big.NewInt(10),
		},
	}
	data, err := candidateList.Serialize()
	require.NoError(err)
	chainClient.EXPECT().ReadState(gomock.Any(), readStateRequest).Times(1).Return(&iotexapi.ReadStateResponse{
		Data: data,
	}, nil)

	blk, err := testutil.BuildCompleteBlock(uint64(180), uint64(361))
	require.NoError(err)

	require.NoError(store.Transact(func(tx *sql.Tx) error {
		return bp.HandleBlock(bpctx, tx, blk)
	}))

	require.NoError(store.Transact(func(tx *sql.Tx) error {
		return p.HandleBlock(ctx, tx, blk)
	}))

	actionHash := blk.Actions[6].Hash()
	receiptHash := blk.Receipts[6].Hash()
	xrc20History, err := p.getXrc20History("xxxxx")
	require.NoError(err)

	require.Equal(hex.EncodeToString(actionHash[:]), xrc20History[0].ActionHash)
	require.Equal(hex.EncodeToString(receiptHash[:]), xrc20History[0].ReceiptHash)
	require.Equal("xxxxx", xrc20History[0].Address)

	require.Equal(transferSha3, xrc20History[0].Topics)
	require.Equal("0000000000000000000000006356908ace09268130dee2b7de643314bbeb3683000000000000000000000000da7e12ef57c236a06117c5e0d04a228e7181cf360000000000000000000000000000000000000000000000000de0b6b3a7640000", xrc20History[0].Data)
	require.Equal("100000", xrc20History[0].BlockHeight)
	require.Equal("888", xrc20History[0].Index)
	require.Equal("failure", xrc20History[0].Status)
}

func TestCheckIsErc20(t *testing.T) {
	chainEndpoint := "api.testnet.iotex.one:80"
	grpcCtx1, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	conn1, err := grpc.DialContext(grpcCtx1, chainEndpoint, grpc.WithBlock(), grpc.WithInsecure())
	if err != nil {
		log.L().Error("Failed to connect to chain's API server.")
	}

	chainClient := iotexapi.NewAPIServiceClient(conn1)

	ctx := indexcontext.WithIndexCtx(context.Background(), indexcontext.IndexCtx{
		ChainClient: chainClient,
	})
	r := checkIsErc20(ctx, "io1fpnufwk6j4fjz6ljjmzvvn5l7p6fypjfjwmde8")
	fmt.Println(r)
	fmt.Println("////////////////////////////////")
	r = checkIsErc20(ctx, "io1wg80fjr9jy4kuwcq7j5ujyq7m0akgqg9vzgymp")
	fmt.Println(r)
	// normal address,not contract
	r = checkIsErc20(ctx, "io1ph0u2psnd7muq5xv9623rmxdsxc4uapxhzpg02")
	fmt.Println(r)
}

func checkIsErc20(ctx context.Context, addr string) bool {
	if _, ok := notxrc20contract[addr]; ok {
		fmt.Println("cache have")
		return false
	}
	if _, ok := xrc20contract[addr]; ok {
		fmt.Println("cache have")
		return true
	}
	indexCtx := indexcontext.MustGetIndexCtx(ctx)
	if indexCtx.ChainClient == nil {
		return false
	}
	ret := readContract(indexCtx.ChainClient, addr, 1, totalSupply)
	if !ret {
		return false
	}

	ret = readContract(indexCtx.ChainClient, addr, 2, balanceOf)
	if !ret {
		return false
	}
	ret = readContract(indexCtx.ChainClient, addr, 3, allowance)
	if !ret {
		return false
	}
	ret = readContract(indexCtx.ChainClient, addr, 5, approve)
	if !ret {
		return false
	}
	xrc20contract[addr] = struct{}{}
	return true
	//check transfer and transferFrom is not nessessary,because those two's results are the same as the contract that have no such function
	//ret = readContract(indexCtx.ChainClient, addr, 4, transfer)
	//if !ret {
	//	fmt.Println("transfer")
	//	return false
	//}
	//
	//return readContract(indexCtx.ChainClient, addr, 6, transferFrom)
}
