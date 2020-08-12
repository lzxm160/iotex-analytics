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
}

func TestGetAllStakingCandidates(t *testing.T) {
	require := require.New(t)
	grpcCtx1, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	conn1, err := grpc.DialContext(grpcCtx1, "api.iotex.one:80", grpc.WithBlock(), grpc.WithInsecure())
	require.NoError(err)
	chainClient := iotexapi.NewAPIServiceClient(conn1)
	resp, err := GetAllStakingCandidates(chainClient, 6360121)
	require.NoError(err)
	for _, r := range resp.GetCandidates() {
		fmt.Println(r)
	}
}
func TestGetAllStakingBuckets(t *testing.T) {
	require := require.New(t)
	grpcCtx1, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	conn1, err := grpc.DialContext(grpcCtx1, "api.iotex.one:80", grpc.WithBlock(), grpc.WithInsecure())
	require.NoError(err)
	chainClient := iotexapi.NewAPIServiceClient(conn1)
	resp, err := GetAllStakingBuckets(chainClient, 6360121)
	require.NoError(err)
	for _, r := range resp.GetBuckets() {
		fmt.Println(r)
	}
}
