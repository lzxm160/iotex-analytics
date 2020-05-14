// Copyright (c) 2020 IoTeX
// This is an alpha (internal) release and is not suitable for production. This source code is provided 'as is' and no
// warranties are given as to title or non-infringement, merchantability or fitness for purpose and, to the extent
// permitted by law, all liability for your use of the code is disclaimed. This source code is governed by Apache
// License 2.0 that can be found in the LICENSE file.

package indexprotocol

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/iotexproject/iotex-core/pkg/log"
	"github.com/iotexproject/iotex-proto/golang/iotexapi"
	"google.golang.org/grpc"

	"github.com/stretchr/testify/require"
)

const candidateName = "726f626f7462703030303030"

func TestEnDecodeName(t *testing.T) {
	require := require.New(t)
	decoded, err := DecodeDelegateName(candidateName)
	require.NoError(err)

	encoded, err := EncodeDelegateName(decoded)
	require.NoError(err)
	require.Equal(candidateName, encoded)

	/////////////////////////////////
	chainEndpoint := "35.236.100.38:14014"
	grpcCtx1, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	conn1, err := grpc.DialContext(grpcCtx1, chainEndpoint, grpc.WithBlock(), grpc.WithInsecure())
	if err != nil {
		log.L().Error("Failed to connect to chain's API server.")
	}
	chainClient := iotexapi.NewAPIServiceClient(conn1)
	cand, err := GetAllStakingCandidates(chainClient, 0)
	require.NoError(err)
	for _, c := range cand.Candidates {
		fmt.Println(c)
	}

	fmt.Println("/////////////////////////////////")
	buckets, err := GetAllStakingBuckets(chainClient, 0)
	require.NoError(err)
	for _, c := range buckets.Buckets {
		fmt.Println(c)
	}
}
