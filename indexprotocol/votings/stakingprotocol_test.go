// Copyright (c) 2020 IoTeX
// This is an alpha (internal) release and is not suitable for production. This source code is provided 'as is' and no
// warranties are given as to title or non-infringement, merchantability or fitness for purpose and, to the extent
// permitted by law, all liability for your use of the code is disclaimed. This source code is governed by Apache
// License 2.0 that can be found in the LICENSE file.

package votings

import (
	"context"
	"encoding/hex"
	"fmt"
	"math/big"
	"strconv"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/timestamp"
	"github.com/stretchr/testify/require"

	"github.com/iotexproject/iotex-core/ioctl/util"
	"github.com/iotexproject/iotex-core/test/mock/mock_apiserviceclient"
	"github.com/iotexproject/iotex-proto/golang/iotexapi"
	"github.com/iotexproject/iotex-proto/golang/iotextypes"

	"github.com/iotexproject/iotex-analytics/epochctx"
	"github.com/iotexproject/iotex-analytics/indexprotocol"
	s "github.com/iotexproject/iotex-analytics/sql"
	"github.com/iotexproject/iotex-analytics/testutil"
)

var (
	now     = time.Now()
	buckets = []*iotextypes.VoteBucket{
		&iotextypes.VoteBucket{
			Index:            10,
			CandidateAddress: "io1mflp9m6hcgm2qcghchsdqj3z3eccrnekx9p0ms",
			StakedAmount:     "30000",
			StakedDuration:   1, // one day
			CreateTime:       &timestamp.Timestamp{Seconds: now.Unix(), Nanos: 0},
			StakeStartTime:   &timestamp.Timestamp{Seconds: now.Unix(), Nanos: 0},
			UnstakeStartTime: &timestamp.Timestamp{Seconds: now.Unix(), Nanos: 0},
			AutoStake:        false,
			Owner:            "io1l9vaqmanwj47tlrpv6etf3pwq0s0snsq4vxke2",
		},
		&iotextypes.VoteBucket{
			Index:            11,
			CandidateAddress: "io1mflp9m6hcgm2qcghchsdqj3z3eccrnekx9p0ms",
			StakedAmount:     "30000",
			StakedDuration:   1,
			CreateTime:       &timestamp.Timestamp{Seconds: now.Unix(), Nanos: 0},
			StakeStartTime:   &timestamp.Timestamp{Seconds: now.Unix(), Nanos: 0},
			UnstakeStartTime: &timestamp.Timestamp{Seconds: now.Unix(), Nanos: 0},
			AutoStake:        false,
			Owner:            "io1ph0u2psnd7muq5xv9623rmxdsxc4uapxhzpg02",
		},
		&iotextypes.VoteBucket{
			Index:            12,
			CandidateAddress: "io1mflp9m6hcgm2qcghchsdqj3z3eccrnekx9p0ms",
			StakedAmount:     "30000",
			StakedDuration:   1,
			CreateTime:       &timestamp.Timestamp{Seconds: now.Unix(), Nanos: 0},
			StakeStartTime:   &timestamp.Timestamp{Seconds: now.Unix(), Nanos: 0},
			UnstakeStartTime: &timestamp.Timestamp{Seconds: now.Unix(), Nanos: 0},
			AutoStake:        false,
			Owner:            "io1vdtfpzkwpyngzvx7u2mauepnzja7kd5rryp0sg",
		},
	}
	candidates = []*iotextypes.CandidateV2{
		&iotextypes.CandidateV2{
			OwnerAddress:       "io1mflp9m6hcgm2qcghchsdqj3z3eccrnekx9p0ms",
			OperatorAddress:    "io1mflp9m6hcgm2qcghchsdqj3z3eccrnekx9p0ms",
			RewardAddress:      "io1mflp9m6hcgm2qcghchsdqj3z3eccrnekx9p0ms",
			Name:               delegateName,
			TotalWeightedVotes: "10",
			SelfStakeBucketIdx: 6666,
			SelfStakingTokens:  "99999",
		},
	}
	delegateName = "xxxx"
)

func TestStaking(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	chainClient := mock_apiserviceclient.NewMockServiceClient(ctrl)
	mock(chainClient, t)
	height := uint64(110000)
	epochNumber := uint64(68888)
	require := require.New(t)
	ctx := context.Background()
	//use for remote database
	testutil.CleanupDatabase(t, connectStr, dbName)
	store := s.NewMySQL(connectStr, dbName)
	require.NoError(store.Start(ctx))
	defer func() {
		//use for remote database
		_, err := store.GetDB().Exec("DROP DATABASE " + dbName)
		require.NoError(err)
		require.NoError(store.Stop(ctx))
	}()
	require.NoError(store.Start(context.Background()))
	cfg := indexprotocol.VoteWeightCalConsts{
		DurationLg: 1.2,
		AutoStake:  1,
		SelfStake:  1.06,
	}
	p, err := NewProtocol(store, epochctx.NewEpochCtx(36, 24, 15, epochctx.FairbankHeight(100)), indexprotocol.GravityChain{}, indexprotocol.Poll{
		VoteThreshold:        "100000000000000000000",
		ScoreThreshold:       "0",
		SelfStakingThreshold: "0",
	}, cfg)
	require.NoError(err)
	require.NoError(p.CreateTables(context.Background()))
	tx, err := p.Store.GetDB().Begin()
	require.NoError(err)
	require.NoError(p.processStaking(tx, chainClient, height, epochNumber, nil, 0))
	require.NoError(tx.Commit())
	// case I: checkout bucket if it's written right
	require.NoError(err)
	ret, err := p.stakingBucketTableOperator.Get(height, p.Store.GetDB(), nil)
	require.NoError(err)
	bucketList, ok := ret.(*iotextypes.VoteBucketList)
	require.True(ok)
	bucketsBytes, _ := proto.Marshal(&iotextypes.VoteBucketList{Buckets: buckets})
	bucketListBytes, _ := proto.Marshal(bucketList)
	require.EqualValues(bucketsBytes, bucketListBytes)

	// case II: checkout candidate if it's written right
	ret, err = p.stakingCandidateTableOperator.Get(height, p.Store.GetDB(), nil)
	require.NoError(err)
	candidateList, ok := ret.(*iotextypes.CandidateListV2)
	require.True(ok)
	require.Equal(delegateName, candidateList.Candidates[0].Name)
	candidatesBytes, _ := proto.Marshal(&iotextypes.CandidateListV2{Candidates: candidates})
	candidateListBytes, _ := proto.Marshal(candidateList)
	require.EqualValues(candidatesBytes, candidateListBytes)

	// case III: check getStakingBucketInfoByEpoch
	encodedName, err := indexprotocol.EncodeDelegateName(delegateName)
	require.NoError(err)
	bucketInfo, err := p.getStakingBucketInfoByEpoch(height, epochNumber, encodedName)
	require.NoError(err)

	ethAddress1, err := util.IoAddrToEvmAddr("io1l9vaqmanwj47tlrpv6etf3pwq0s0snsq4vxke2")
	require.NoError(err)
	require.Equal(hex.EncodeToString(ethAddress1.Bytes()), bucketInfo[0].VoterAddress)

	ethAddress2, err := util.IoAddrToEvmAddr("io1ph0u2psnd7muq5xv9623rmxdsxc4uapxhzpg02")
	require.NoError(err)
	require.Equal(hex.EncodeToString(ethAddress2.Bytes()), bucketInfo[1].VoterAddress)

	ethAddress3, err := util.IoAddrToEvmAddr("io1vdtfpzkwpyngzvx7u2mauepnzja7kd5rryp0sg")
	require.NoError(err)
	require.Equal(hex.EncodeToString(ethAddress3.Bytes()), bucketInfo[2].VoterAddress)
	for _, b := range bucketInfo {
		require.True(b.Decay)
		require.Equal(epochNumber, b.EpochNumber)
		require.True(b.IsNative)
		dur, err := time.ParseDuration(b.RemainingDuration)
		require.NoError(err)
		require.True(dur.Seconds() <= float64(86400))
		// 'now' need to format b/c b.StartTime's nano time is set to 0
		require.Equal(now.Format("2006-01-02 15:04:05 -0700 MST"), b.StartTime)
		require.Equal("30000", b.Votes)
		require.Equal("30000", b.WeightedVotes)
	}
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
		AutoStake:      false,
	}
	remaining = remainingTime(bucket)
	require.True(remaining > 0 && remaining < time.Duration(100*24*time.Hour))

	// case III: AutoStake is true
	bucketTime = time.Unix(time.Now().Unix()-10, 0)
	timestamp, _ = ptypes.TimestampProto(bucketTime)
	bucket = &iotextypes.VoteBucket{
		StakeStartTime: timestamp,
		StakedDuration: 100,
		AutoStake:      true,
	}
	remaining = remainingTime(bucket)
	require.Equal(time.Duration(100*24*time.Hour), remaining)

	// case IV: now is after starttime+stakedduration
	bucketTime = time.Unix(time.Now().Unix()-86410, 0)
	timestamp, _ = ptypes.TimestampProto(bucketTime)
	bucket = &iotextypes.VoteBucket{
		StakeStartTime: timestamp,
		StakedDuration: 1,
	}
	remaining = remainingTime(bucket)
	require.Equal(time.Duration(0), remaining)
}

func TestFilterStakingCandidates(t *testing.T) {
	require := require.New(t)
	cl := &iotextypes.CandidateListV2{Candidates: candidates}
	unqualifiedList := &iotextypes.ProbationCandidateList{
		IntensityRate: 10,
		ProbationList: []*iotextypes.ProbationCandidateList_Info{
			&iotextypes.ProbationCandidateList_Info{
				Address: "io1mflp9m6hcgm2qcghchsdqj3z3eccrnekx9p0ms",
				Count:   10,
			},
		},
	}
	cl, err := filterStakingCandidates(cl, unqualifiedList, 10)
	require.NoError(err)
	require.Equal("9", cl.Candidates[0].TotalWeightedVotes)
}

func mock(chainClient *mock_apiserviceclient.MockServiceClient, t *testing.T) {
	protocolID := "staking"
	readBucketsLimit := uint32(30000)
	readCandidatesLimit := uint32(20000)
	require := require.New(t)
	methodNameBytes, _ := proto.Marshal(&iotexapi.ReadStakingDataMethod{
		Method: iotexapi.ReadStakingDataMethod_BUCKETS,
	})
	arg, err := proto.Marshal(&iotexapi.ReadStakingDataRequest{
		Request: &iotexapi.ReadStakingDataRequest_Buckets{
			Buckets: &iotexapi.ReadStakingDataRequest_VoteBuckets{
				Pagination: &iotexapi.PaginationParam{
					Offset: 0,
					Limit:  readBucketsLimit,
				},
			},
		},
	})
	readStateRequest := &iotexapi.ReadStateRequest{
		ProtocolID: []byte(protocolID),
		MethodName: methodNameBytes,
		Arguments:  [][]byte{arg, []byte(strconv.FormatUint(110000, 10))},
	}

	vbl := &iotextypes.VoteBucketList{Buckets: buckets}
	s, err := proto.Marshal(vbl)
	require.NoError(err)
	first := chainClient.EXPECT().ReadState(gomock.Any(), readStateRequest).AnyTimes().Return(&iotexapi.ReadStateResponse{
		Data: s,
	}, nil)

	// mock candidate
	methodNameBytes, _ = proto.Marshal(&iotexapi.ReadStakingDataMethod{
		Method: iotexapi.ReadStakingDataMethod_CANDIDATES,
	})
	arg, err = proto.Marshal(&iotexapi.ReadStakingDataRequest{
		Request: &iotexapi.ReadStakingDataRequest_Candidates_{
			Candidates: &iotexapi.ReadStakingDataRequest_Candidates{
				Pagination: &iotexapi.PaginationParam{
					Offset: 0,
					Limit:  readCandidatesLimit,
				},
			},
		},
	})
	readStateRequest = &iotexapi.ReadStateRequest{
		ProtocolID: []byte(protocolID),
		MethodName: methodNameBytes,
		Arguments:  [][]byte{arg, []byte(strconv.FormatUint(110000, 10))},
	}

	cl := &iotextypes.CandidateListV2{Candidates: candidates}
	s, err = proto.Marshal(cl)
	require.NoError(err)
	second := chainClient.EXPECT().ReadState(gomock.Any(), readStateRequest).AnyTimes().Return(&iotexapi.ReadStateResponse{
		Data: s,
	}, nil)
	third := chainClient.EXPECT().ReadState(gomock.Any(), gomock.Any()).AnyTimes().Return(&iotexapi.ReadStateResponse{
		Data: []byte("888888888"),
	}, nil)
	gomock.InOrder(
		first,
		second,
		third,
	)
}

func getcandidateTotal(name string, t *testing.T, local bool) *big.Int {
	require := require.New(t)
	ctx := context.Background()
	var store s.Store
	if local {
		store = s.NewMySQL("root:123456@tcp(192.168.146.140:3306)/", "analytics")
	} else {
		store = s.NewMySQL("admin:IOTX4689@tcp(analytics-testnet.cs4r6igeqju2.us-east-1.rds.amazonaws.com:3306)/", "analytics")
	}

	require.NoError(store.Start(ctx))
	defer func() {
		require.NoError(store.Stop(ctx))
	}()
	require.NoError(store.Start(context.Background()))
	cfg := indexprotocol.VoteWeightCalConsts{
		DurationLg: 1.2,
		AutoStake:  1,
		SelfStake:  1.06,
	}
	p, err := NewProtocol(store, epochctx.NewEpochCtx(36, 24, 15, epochctx.FairbankHeight(3252241), epochctx.EnableDardanellesSubEpoch(100081, 30)), indexprotocol.GravityChain{}, indexprotocol.Poll{
		VoteThreshold:        "100000000000000000000",
		ScoreThreshold:       "0",
		SelfStakingThreshold: "0",
	}, cfg)
	require.NoError(err)
	//epoch := uint64(4665)
	epoch := uint64(4670)
	startHeight := p.epochCtx.GetEpochHeight(epoch)
	fmt.Println(epoch, startHeight)
	can, err := p.stakingCandidateTableOperator.Get(startHeight, p.Store.GetDB(), nil)
	require.NoError(err)
	candidateList, ok := can.(*iotextypes.CandidateListV2)
	require.True(ok)
	twv := big.NewInt(0)
	fmt.Println(name, hex.EncodeToString([]byte(name)))
	for _, cand := range candidateList.Candidates {
		encodedName, err := indexprotocol.EncodeDelegateName(cand.Name)
		require.NoError(err)
		if encodedName == hex.EncodeToString([]byte(name)) {
			twv.SetString(cand.TotalWeightedVotes, 10)
			return twv
		}
	}
	return big.NewInt(0)
}
func TestVotes(t *testing.T) {
	candidatesTotal := getcandidateTotal("robotbp00021", t, true)
	fmt.Println("candidatesTotal", candidatesTotal.String())

	// for"robotbp00021"
	//726f626f7462703030303231
	//select * from aggregate_voting where candidate_name='726f626f7462703030303231' and epoch_number=4665
	a, _ := big.NewInt(0).SetString("11447588583684619435300000", 10)
	b, _ := big.NewInt(0).SetString("2289494821788706000000000", 10)
	c, _ := big.NewInt(0).SetString("228949482178870600000", 10)
	d := a.Add(a, b).Add(a, c)
	fmt.Println("local aggregate_voting mysql", d.String())
	fmt.Println("local staking_candidate more than aggregate_voting sum", candidatesTotal.Sub(candidatesTotal, d).String())

	amount, ok := new(big.Float).SetString("10000000000000000000000000")
	if !ok {
		fmt.Println("!ok")
	}
	weightedAmount, acc := amount.Mul(amount, big.NewFloat(1.06)).Int(nil)
	fmt.Println(weightedAmount.String(), acc)
	//	//
	//	////remote
	//	//
	//	//remotea, _ := big.NewInt(0).SetString("10380282203476409510403440", 10)
	//	//remoteb, _ := big.NewInt(0).SetString("2200597821158787326148608", 10)
	//	//remotec, _ := big.NewInt(0).SetString("207603568033847851744", 10)
	//	//remoted := remotea.Add(remotea, remoteb).Add(remotea, remotec)
	//	//fmt.Println("remote", remoted.String())
	//	//fmt.Println("remote aggregate_voting sum less than staking_candidate", remotetwv.Sub(remotetwv, remoted).String())

	// for"robotbp00012"
	//726f626f7462703030303132
	//select * from aggregate_voting where candidate_name='726f626f7462703030303132' and epoch_number=4665
	//a, _ := big.NewInt(0).SetString("2289494821788705003536384", 10)
	//fmt.Println("local", a.String())
	//fmt.Println("local aggregate_voting sum less than staking_candidate", candidatesTotal.Sub(candidatesTotal, a).String())
	//votingPower := new(big.Float).SetInt(a)
	//val, _ := votingPower.Mul(votingPower, big.NewFloat(90)).Int(nil)
	//fmt.Println("val", val.String())
}

//func calculateVoteWeight2(cfg indexprotocol.VoteWeightCalConsts, duration uint32, selfStake bool) *big.Int {
//	remainingTime := float64(duration * 86400)
//	weight := float64(1)
//	var m float64
//	if true {
//		m = cfg.AutoStake
//	}
//	if remainingTime > 0 {
//		weight += math.Log(math.Ceil(remainingTime/86400)*(1+m)) / math.Log(cfg.DurationLg) / 100
//	}
//	if selfStake && true && duration >= 91 {
//		// self-stake extra bonus requires enable auto-stake for at least 3 months
//		weight *= cfg.SelfStake
//	}
//	amount, ok := new(big.Float).SetString(v.StakedAmount)
//	if !ok {
//		return big.NewInt(0)
//	}
//	weightedAmount, _ := amount.Mul(amount, big.NewFloat(weight)).Int(nil)
//	return weightedAmount
//}
