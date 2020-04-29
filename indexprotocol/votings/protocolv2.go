// Copyright (c) 2020 IoTeX
// This is an alpha (internal) release and is not suitable for production. This source code is provided 'as is' and no
// warranties are given as to title or non-infringement, merchantability or fitness for purpose and, to the extent
// permitted by law, all liability for your use of the code is disclaimed. This source code is governed by Apache
// License 2.0 that can be found in the LICENSE file.

package votings

import (
	"context"
	"database/sql"
	"encoding/hex"
	"fmt"
	"math"
	"math/big"

	"github.com/golang/protobuf/proto"
	"github.com/pkg/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/iotexproject/iotex-address/address"
	"github.com/iotexproject/iotex-proto/golang/iotexapi"
	"github.com/iotexproject/iotex-proto/golang/iotextypes"

	"github.com/iotexproject/iotex-analytics/indexprotocol"
)

const (
	protocolID = "staking"
)

func (p *Protocol) stakingV2(chainClient iotexapi.APIServiceClient, epochStartheight, epochNumber uint64, probationList *iotextypes.ProbationCandidateList) (err error) {
	fmt.Println("stakingv2:", epochNumber)
	bucketsCount, err := p.getBucketsCountV2(chainClient)
	if err != nil {
		return errors.Wrap(err, "failed to get buckets count")
	}
	candidatesCount, err := p.getCandidatesCountV2(chainClient)
	if err != nil {
		return errors.Wrap(err, "failed to get candidates count")
	}
	voteBucketList, err := p.getBucketsV2(chainClient, 0, bucketsCount)
	if err != nil {
		return errors.Wrap(err, "failed to get bucket")
	}
	candidateList, err := p.getCandidatesV2(chainClient, 0, candidatesCount)
	if err != nil {
		return errors.Wrap(err, "failed to get bucket")
	}
	if probationList != nil {
		err = filterCandidatesV2(candidateList, probationList)
		if err != nil {
			return errors.Wrap(err, "failed to filter candidate with probation list")
		}
	}
	// after get and clean data,the following code is for writing mysql
	tx, err := p.Store.GetDB().Begin()
	// update stakingV2_bucket and height_to_stakingV2_bucket table
	if err = p.nativeV2BucketTableOperator.Put(epochStartheight, voteBucketList, tx); err != nil {
		return
	}
	// update stakingV2_candidate and height_to_stakingV2_candidate table
	if err = p.nativeV2CandidateTableOperator.Put(epochStartheight, candidateList, tx); err != nil {
		return
	}
	// update voting_result table
	if err = p.updateVotingResultV2(tx, candidateList, epochNumber); err != nil {
		return
	}
	// call aggregate_voting and voting_meta table
	if err = p.updateAggregateVotingV2(tx, voteBucketList, candidateList, epochNumber, probationList); err != nil {
		return
	}
	tx.Commit()
	return
}

func (p *Protocol) getBucketsCountV2(chainClient iotexapi.APIServiceClient) (count uint32, err error) {
	// TODO waiting for iotex-core's api
	count = uint32(10)
	return
}

func (p *Protocol) getCandidatesCountV2(chainClient iotexapi.APIServiceClient) (count uint32, err error) {
	// TODO waiting for iotex-core's api
	count = uint32(10)
	return
}

func (p *Protocol) getBucketsV2(chainClient iotexapi.APIServiceClient, offset, limit uint32) (voteBucketList *iotextypes.VoteBucketList, err error) {
	methodName := &iotexapi.ReadStakingDataMethod{
		Method: iotexapi.ReadStakingDataMethod_BUCKETS,
	}
	methodNameBytes, _ := proto.Marshal(methodName)
	arguments := &iotexapi.ReadStakingDataRequest_VoteBuckets{
		Pagination: &iotexapi.PaginationParam{
			Offset: offset,
			Limit:  limit,
		},
	}
	argumentsBytes, _ := proto.Marshal(arguments)
	readStateRequest := &iotexapi.ReadStateRequest{
		ProtocolID: []byte(protocolID),
		MethodName: methodNameBytes,
		Arguments:  [][]byte{argumentsBytes},
	}
	readStateRes, err := chainClient.ReadState(context.Background(), readStateRequest)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			fmt.Println("ReadStakingDataMethod_BUCKETS not found")
			return
		}
		return
	}
	voteBucketList = &iotextypes.VoteBucketList{}
	if err := proto.Unmarshal(readStateRes.GetData(), voteBucketList); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal VoteBucketList")
	}
	return
}

func (p *Protocol) getCandidatesV2(chainClient iotexapi.APIServiceClient, offset, limit uint32) (candidateList *iotextypes.CandidateListV2, err error) {
	methodName := &iotexapi.ReadStakingDataMethod{
		Method: iotexapi.ReadStakingDataMethod_CANDIDATES,
	}
	methodNameBytes, _ := proto.Marshal(methodName)
	arguments := &iotexapi.ReadStakingDataRequest_Candidates{
		Pagination: &iotexapi.PaginationParam{
			Offset: offset,
			Limit:  limit,
		},
	}
	argumentsBytes, _ := proto.Marshal(arguments)
	readStateRequest := &iotexapi.ReadStateRequest{
		ProtocolID: []byte(protocolID),
		MethodName: methodNameBytes,
		Arguments:  [][]byte{argumentsBytes},
	}
	readStateRes, err := chainClient.ReadState(context.Background(), readStateRequest)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			fmt.Println("ReadStakingDataMethod_BUCKETS not found")
			return
		}
		return
	}
	candidateList = &iotextypes.CandidateListV2{}
	if err := proto.Unmarshal(readStateRes.GetData(), candidateList); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal VoteBucketList")
	}
	return
}

func (p *Protocol) updateVotingResultV2(tx *sql.Tx, candidates *iotextypes.CandidateListV2, epochNumber uint64) (err error) {
	var voteResultStmt *sql.Stmt
	insertQuery := fmt.Sprintf(insertVotingResult,
		VotingResultTableName)
	if voteResultStmt, err = tx.Prepare(insertQuery); err != nil {
		return err
	}
	defer func() {
		closeErr := voteResultStmt.Close()
		if err == nil && closeErr != nil {
			err = closeErr
		}
	}()
	for _, candidate := range candidates.Candidates {
		addr, err := address.FromString(candidate.OwnerAddress)
		if err != nil {
			return err
		}
		addressString := hex.EncodeToString(addr.Bytes())
		if _, err = voteResultStmt.Exec(
			epochNumber,
			candidate.Name,
			candidate.OperatorAddress,
			candidate.RewardAddress,
			candidate.TotalWeightedVotes,
			candidate.SelfStakingTokens,
			0,             // TODO wait for research
			0,             // TODO wait for research
			0,             // TODO wait for research
			addressString, // type is varchar 40,change to ethereum hex address
		); err != nil {
			return err
		}
	}
	return nil
}

func (p *Protocol) updateAggregateVotingV2(tx *sql.Tx, votes *iotextypes.VoteBucketList, delegates *iotextypes.CandidateListV2, epochNumber uint64, probationList *iotextypes.ProbationCandidateList) (err error) {
	// TODO check if it's need to filter probation again for staking v2
	probationMap := make(map[string]uint32)
	var intensityRate float64
	if probationList != nil {
		intensityRate = float64(uint32(100)-probationList.IntensityRate) / float64(100)
		for _, delegate := range delegates.Candidates {
			for _, elem := range probationList.ProbationList {
				if elem.Address == delegate.OperatorAddress {
					probationMap[delegate.Name] = elem.Count
				}
			}
		}
	}
	//update aggregate voting table
	sumOfWeightedVotes := make(map[aggregateKey]*big.Int)
	totalVoted := big.NewInt(0)
	for _, vote := range votes.Buckets {
		//for sumOfWeightedVotes
		key := aggregateKey{
			epochNumber:   epochNumber,
			candidateName: vote.CandidateAddress,
			voterAddress:  vote.Owner,
			isNative:      true, //alway true for staking v2,TODO check if it's right
		}
		// TODO check if it's right
		weightedAmount := calculateVoteWeightV2(p.voteCfg, vote, false)
		stakeAmount, ok := big.NewInt(0).SetString(vote.StakedAmount, 10)
		if !ok {
			err = errors.New("stake amount convert error")
			return
		}
		if val, ok := sumOfWeightedVotes[key]; ok {
			val.Add(val, weightedAmount)
		} else {
			sumOfWeightedVotes[key] = weightedAmount
		}
		totalVoted.Add(totalVoted, stakeAmount)
	}
	insertQuery := fmt.Sprintf(insertAggregateVoting, AggregateVotingTableName)
	var aggregateStmt *sql.Stmt
	if aggregateStmt, err = tx.Prepare(insertQuery); err != nil {
		return err
	}
	defer func() {
		closeErr := aggregateStmt.Close()
		if err == nil && closeErr != nil {
			err = closeErr
		}
	}()
	for key, val := range sumOfWeightedVotes {
		if _, ok := probationMap[key.candidateName]; ok {
			// filter based on probation
			votingPower := new(big.Float).SetInt(val)
			val, _ = votingPower.Mul(votingPower, big.NewFloat(intensityRate)).Int(nil)
		}
		if _, err = aggregateStmt.Exec(
			key.epochNumber,
			key.candidateName,
			key.voterAddress,
			key.isNative,
			val.Text(10),
		); err != nil {
			return err
		}
	}
	//update voting meta table
	totalWeighted := big.NewInt(0)
	for _, cand := range delegates.Candidates {
		totalWeightedVotes, ok := big.NewInt(0).SetString(cand.TotalWeightedVotes, 10)
		if !ok {
			err = errors.New("total weighted votes convert error")
			return
		}
		totalWeighted.Add(totalWeighted, totalWeightedVotes)
	}
	insertQuery = fmt.Sprintf(insertVotingMeta, VotingMetaTableName)
	if _, err = tx.Exec(insertQuery,
		epochNumber,
		totalVoted.Text(10),
		len(delegates.Candidates),
		totalWeighted.Text(10),
	); err != nil {
		return errors.Wrap(err, "failed to update voting meta table")
	}
	return
}

func calculateVoteWeightV2(cfg indexprotocol.VoteWeightCalConsts, v *iotextypes.VoteBucket, selfStake bool) *big.Int {
	remainingTime := float64(v.StakedDuration)
	weight := float64(1)
	var m float64
	if v.AutoStake {
		m = cfg.AutoStake
	}
	if remainingTime > 0 {
		weight += math.Log(math.Ceil(remainingTime/86400)*(1+m)) / math.Log(cfg.DurationLg) / 100
	}
	if selfStake {
		weight *= cfg.SelfStake
	}

	amount, ok := new(big.Float).SetString(v.StakedAmount)
	if !ok {
		return big.NewInt(0)
	}
	weightedAmount, _ := amount.Mul(amount, big.NewFloat(weight)).Int(nil)
	return weightedAmount
}

// filterCandidatesV2 returns filtered candidate list by given raw candidate and probation list
func filterCandidatesV2(
	candidates *iotextypes.CandidateListV2,
	unqualifiedList *iotextypes.ProbationCandidateList,
) (err error) {
	// TODO check if this algrithom right for staking v2
	intensityRate := float64(uint32(100)-unqualifiedList.IntensityRate) / float64(100)
	probationMap := make(map[string]uint32)
	for _, elem := range unqualifiedList.ProbationList {
		probationMap[elem.Address] = elem.Count
	}
	for i, cand := range candidates.Candidates {
		if _, ok := probationMap[cand.OperatorAddress]; ok {
			// if it is an unqualified delegate, multiply the voting power with probation intensity rate
			votingPower, ok := new(big.Float).SetString(cand.TotalWeightedVotes)
			if !ok {
				return errors.New("total weighted votes convert error")
			}
			newVotingPower, _ := votingPower.Mul(votingPower, big.NewFloat(intensityRate)).Int(nil)
			candidates.Candidates[i].TotalWeightedVotes = newVotingPower.String()
		}
	}
	return nil
}
