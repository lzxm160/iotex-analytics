package votings

import (
	"context"
	"fmt"
	"testing"

	"github.com/golang/protobuf/ptypes/timestamp"
	"github.com/stretchr/testify/require"

	"github.com/iotexproject/iotex-election/committee"

	"github.com/iotexproject/iotex-analytics/epochctx"
	"github.com/iotexproject/iotex-analytics/indexprotocol"
	staking "github.com/iotexproject/iotex-analytics/indexprotocol/votings/stakingpb"
	s "github.com/iotexproject/iotex-analytics/sql"
)

const (
	localconnectStr = "root:123456@tcp(192.168.146.140:3306)/"
	localdbName     = "analytics"
)

func TestXX(t *testing.T) {
	require := require.New(t)
	store := s.NewMySQL(localconnectStr, localdbName)
	require.NoError(store.Start(context.Background()))
	p, err := NewProtocol(store, epochctx.NewEpochCtx(36, 24, 15), indexprotocol.GravityChain{}, indexprotocol.Poll{
		VoteThreshold:        "100000000000000000000",
		ScoreThreshold:       "0",
		SelfStakingThreshold: "0",
	})
	require.NoError(err)
	tx, err := p.Store.GetDB().Begin()
	require.NoError(err)
	bucketTableOperator, err := NewBucketTableOperator("stakingV2_bucket", committee.MYSQL)
	require.NoError(err)
	require.NoError(bucketTableOperator.CreateTables(tx))

	buckets := []*staking.Bucket{
		&staking.Bucket{
			Index:            10,
			CandidateAddress: "io1mflp9m6hcgm2qcghchsdqj3z3eccrnekx9p0ms",
			StakedAmount:     "20000",
			StakedDuration:   20,
			CreateTime:       &timestamp.Timestamp{Seconds: int64(1587864599), Nanos: int32(456)},
			StakeStartTime:   &timestamp.Timestamp{Seconds: int64(1587864599), Nanos: int32(123)},
			UnstakeStartTime: &timestamp.Timestamp{Seconds: int64(1587864599), Nanos: int32(321)},
			AutoStake:        false,
			Owner:            "io1mflp9m6hcgm2qcghchsdqj3z3eccrnekx9p0ms",
		},
	}
	require.NoError(bucketTableOperator.Put(3, buckets, tx))
	tx.Commit()
	tx, err = p.Store.GetDB().Begin()
	require.NoError(err)
	ret, err := bucketTableOperator.Get(3, p.Store.GetDB(), tx)
	require.NoError(err)
	buckets, ok := ret.([]*staking.Bucket)
	require.True(ok)
	fmt.Println(buckets[0])

	// test candidate
	fmt.Println("//////////////////////////////")
	tx, err = p.Store.GetDB().Begin()
	require.NoError(err)
	candidateTableOperator, err := NewCandidateTableOperator("stakingV2_candidate", committee.MYSQL)
	require.NoError(err)
	require.NoError(candidateTableOperator.CreateTables(tx))
	candidates := []*staking.Candidate{
		&staking.Candidate{
			OwnerAddress:       "io1mflp9m6hcgm2qcghchsdqj3z3eccrnekx9p0ms",
			OperatorAddress:    "io1mflp9m6hcgm2qcghchsdqj3z3eccrnekx9p0ms",
			RewardAddress:      "io1mflp9m6hcgm2qcghchsdqj3z3eccrnekx9p0ms",
			Name:               "io1mflp9m6hcgm2qcghchsdqj3z3eccrnekx9p0ms",
			Votes:              "5555",
			SelfStakeBucketIdx: 232323,
			SelfStake:          "666666",
		},
	}
	require.NoError(candidateTableOperator.Put(3, candidates, tx))
	tx.Commit()
	tx, err = p.Store.GetDB().Begin()
	require.NoError(err)
	ret, err = candidateTableOperator.Get(3, p.Store.GetDB(), tx)
	require.NoError(err)
	candidates, ok = ret.([]*staking.Candidate)
	require.True(ok)
	fmt.Println(candidates[0])
}
