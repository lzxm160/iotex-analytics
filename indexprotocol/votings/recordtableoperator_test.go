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
			StakedDuration:   10,
			CreateTime:       &timestamp.Timestamp{Seconds: int64(1587864599), Nanos: int32(456)},
			StakeStartTime:   &timestamp.Timestamp{Seconds: int64(1587864599), Nanos: int32(123)},
			UnstakeStartTime: &timestamp.Timestamp{Seconds: int64(1587864599), Nanos: int32(321)},
			AutoStake:        false,
			Owner:            "io1mflp9m6hcgm2qcghchsdqj3z3eccrnekx9p0ms",
		},
	}
	//f, err := InsertVoteBuckets("stakingV2_bucket", committee.MYSQL, buckets, tx)
	//require.NoError(err)
	//for k, v := range f {
	//	fmt.Println(hex.EncodeToString(k[:]), v)
	//}
	defer func() {
		fmt.Println("rollback happens")
		tx.Rollback()
	}()
	require.NoError(bucketTableOperator.Put(2, buckets, tx))
	tx.Commit()
	ret, err := bucketTableOperator.Get(2, p.Store.GetDB(), tx)
	require.NoError(err)
	candidates, ok := ret.([]*staking.Bucket)
	require.True(ok)
	fmt.Println(candidates[0])
}
