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
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/iotexproject/iotex-antenna-go/v2/account"

	"github.com/iotexproject/iotex-antenna-go/v2/iotex"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/iotexproject/iotex-address/address"
	"github.com/iotexproject/iotex-core/pkg/log"
	"go.uber.org/zap"

	"github.com/ethereum/go-ethereum/common"
	"github.com/pkg/errors"

	"github.com/iotexproject/iotex-analytics/contract"
	"github.com/iotexproject/iotex-core/ioctl/util"
	"github.com/iotexproject/iotex-election/db"
	"github.com/iotexproject/iotex-proto/golang/iotexapi"
	"github.com/iotexproject/iotex-proto/golang/iotextypes"

	"github.com/iotexproject/iotex-analytics/indexprotocol"
)

func (p *Protocol) processStaking(tx *sql.Tx, chainClient iotexapi.APIServiceClient, epochStartheight, epochNumber uint64, probationList *iotextypes.ProbationCandidateList) (err error) {
	voteBucketList, err := indexprotocol.GetAllStakingBuckets(chainClient, epochStartheight)
	if err != nil {
		return errors.Wrap(err, "failed to get buckets count")
	}
	candidateList, err := indexprotocol.GetAllStakingCandidates(chainClient, epochStartheight)
	if err != nil {
		return errors.Wrap(err, "failed to get buckets count")
	}
	// after get and clean data,the following code is for writing mysql
	// update staking_bucket and height_to_staking_bucket table
	if err = p.stakingBucketTableOperator.Put(epochStartheight, voteBucketList, tx); err != nil {
		return
	}
	// update staking_candidate and height_to_staking_candidate table
	if err = p.stakingCandidateTableOperator.Put(epochStartheight, candidateList, tx); err != nil {
		return
	}
	if probationList != nil {
		candidateList, err = filterStakingCandidates(candidateList, probationList, epochStartheight)
		if err != nil {
			return errors.Wrap(err, "failed to filter candidate with probation list")
		}
	}
	// update voting_result table
	if err = p.updateStakingResult(tx, candidateList, epochNumber, epochStartheight, chainClient); err != nil {
		return
	}
	// update aggregate_voting and voting_meta table
	if err = p.updateAggregateStaking(tx, voteBucketList, candidateList, epochNumber, probationList); err != nil {
		return
	}
	return
}

func (p *Protocol) updateStakingResult(tx *sql.Tx, candidates *iotextypes.CandidateListV2, epochNumber, epochStartheight uint64, chainClient iotexapi.APIServiceClient) (err error) {
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
		stakingAddress, err := util.IoAddrToEvmAddr(candidate.OwnerAddress)
		if err != nil {
			return errors.Wrap(err, "failed to convert IoTeX address to ETH address")
		}
		blockRewardPortion, epochRewardPortion, foundationBonusPortion, err := p.getStakingDelegateRewardPortions(stakingAddress, epochStartheight, chainClient)
		if err != nil {
			return errors.Errorf("get delegate reward portions:%s,%d,%s", stakingAddress.String(), epochStartheight, err.Error())
		}
		encodedName, err := indexprotocol.EncodeDelegateName(candidate.Name)
		if err != nil {
			return errors.Wrap(err, "encode delegate name error")
		}
		if _, err = voteResultStmt.Exec(
			epochNumber,
			encodedName,
			candidate.OperatorAddress,
			candidate.RewardAddress,
			candidate.TotalWeightedVotes,
			candidate.SelfStakingTokens,
			blockRewardPortion,
			epochRewardPortion,
			foundationBonusPortion,
			hex.EncodeToString(stakingAddress.Bytes()),
		); err != nil {
			return err
		}
	}
	return nil
}

func (p *Protocol) updateAggregateStaking(tx *sql.Tx, votes *iotextypes.VoteBucketList, delegates *iotextypes.CandidateListV2, epochNumber uint64, probationList *iotextypes.ProbationCandidateList) (err error) {
	nameMap, err := ownerAddressToNameMap(delegates)
	if err != nil {
		return errors.Wrap(err, "owner address to name map error")
	}
	pb := convertProbationListToLocal(probationList)
	intensityRate, probationMap := stakingProbationListToMap(delegates, pb)
	//update aggregate voting table
	sumOfWeightedVotes := make(map[aggregateKey]*big.Int)
	totalVoted := big.NewInt(0)
	selfStakeIndex := selfStakeIndexMap(delegates)
	for _, vote := range votes.Buckets {
		//for sumOfWeightedVotes
		key := aggregateKey{
			epochNumber:   epochNumber,
			candidateName: vote.CandidateAddress,
			voterAddress:  vote.Owner,
			isNative:      true,
		}
		selfStake := false
		if _, ok := selfStakeIndex[vote.Index]; ok {
			selfStake = true
		}
		weightedAmount, err := calculateVoteWeight(p.voteCfg, vote, selfStake)
		if err != nil {
			return errors.Wrap(err, "failed to calculate vote weight")
		}
		stakeAmount, ok := big.NewInt(0).SetString(vote.StakedAmount, 10)
		if !ok {
			return errors.New("failed to convert string to big int")
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
		if _, ok := nameMap[key.candidateName]; !ok {
			return errors.New("candidate cannot find name through owner address")
		}
		voterAddress, err := util.IoAddrToEvmAddr(key.voterAddress)
		if err != nil {
			return errors.Wrap(err, "failed to convert IoTeX address to ETH address")
		}
		if _, err = aggregateStmt.Exec(
			key.epochNumber,
			nameMap[key.candidateName],
			hex.EncodeToString(voterAddress.Bytes()),
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

func (p *Protocol) getStakingBucketInfoByEpoch(height, epochNum uint64, delegateName string) ([]*VotingInfo, error) {
	ret, err := p.stakingBucketTableOperator.Get(height, p.Store.GetDB(), nil)
	if errors.Cause(err) == db.ErrNotExist {
		return nil, nil
	}
	if err != nil {
		return nil, errors.Wrap(err, "failed to get staking buckets")
	}
	bucketList, ok := ret.(*iotextypes.VoteBucketList)
	if !ok {
		return nil, errors.Errorf("Unexpected type %s", reflect.TypeOf(ret))
	}
	can, err := p.stakingCandidateTableOperator.Get(height, p.Store.GetDB(), nil)
	if errors.Cause(err) == db.ErrNotExist {
		return nil, nil
	}
	if err != nil {
		return nil, errors.Wrap(err, "failed to get staking candidates")
	}
	candidateList, ok := can.(*iotextypes.CandidateListV2)
	if !ok {
		return nil, errors.Errorf("Unexpected type %s", reflect.TypeOf(can))
	}
	var candidateAddress string
	for _, cand := range candidateList.Candidates {
		encodedName, err := indexprotocol.EncodeDelegateName(cand.Name)
		if err != nil {
			return nil, errors.Wrap(err, "error when encode delegate name")
		}
		if encodedName == delegateName {
			candidateAddress = cand.OwnerAddress
			break
		}
	}
	// update weighted votes based on probation
	pblist, err := p.getProbationList(epochNum)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get probation list from table")
	}
	intensityRate, probationMap := stakingProbationListToMap(candidateList, pblist)
	var votinginfoList []*VotingInfo
	selfStakeIndex := selfStakeIndexMap(candidateList)
	for _, vote := range bucketList.Buckets {
		if vote.CandidateAddress == candidateAddress {
			selfStake := false
			if _, ok := selfStakeIndex[vote.Index]; ok {
				selfStake = true
			}
			weightedVotes, err := calculateVoteWeight(p.voteCfg, vote, selfStake)
			if err != nil {
				return nil, errors.Wrap(err, "calculate vote weight error")
			}
			if _, ok := probationMap[vote.CandidateAddress]; ok {
				// filter based on probation
				votingPower := new(big.Float).SetInt(weightedVotes)
				weightedVotes, _ = votingPower.Mul(votingPower, big.NewFloat(intensityRate)).Int(nil)
			}
			voteOwnerAddress, err := util.IoAddrToEvmAddr(vote.Owner)
			if err != nil {
				return nil, errors.Wrap(err, "failed to convert IoTeX address to ETH address")
			}
			votinginfo := &VotingInfo{
				EpochNumber:       epochNum,
				VoterAddress:      hex.EncodeToString(voteOwnerAddress.Bytes()),
				IsNative:          true,
				Votes:             vote.StakedAmount,
				WeightedVotes:     weightedVotes.Text(10),
				RemainingDuration: remainingTime(vote).String(),
				StartTime:         time.Unix(vote.StakeStartTime.Seconds, int64(vote.StakeStartTime.Nanos)).String(),
				Decay:             !vote.AutoStake,
			}
			votinginfoList = append(votinginfoList, votinginfo)
		}
	}
	return votinginfoList, nil
}

func (p *Protocol) getStakingDelegateRewardPortions(stakingAddress common.Address, epochStartheight uint64, chainClient iotexapi.APIServiceClient) (blockRewardPercentage, epochRewardPercentage, foundationBonusPercentage float64, err error) {
	if p.rewardPortionContract == "" {
		blockRewardPercentage = 100
		epochRewardPercentage = 100
		foundationBonusPercentage = 100
		return
	}
	caddr, err := address.FromString(p.rewardPortionContract)
	if err != nil {
		err = errors.Wrap(err, "Failed to get contract address")
		return
	}
	delegateABI, err := abi.JSON(strings.NewReader(contract.DelegateProfileABI))
	if err != nil {
		err = errors.Wrap(err, "Failed to get parsed delegate profile ABI interface")
		return
	}
	account, err := account.NewAccount()
	if err != nil {
		log.L().Fatal("Failed to create a new account", zap.Error(err))
	}
	c := iotex.NewAuthedClient(chainClient, account)
	// read block reward
	blockRewardPortion, err := c.Contract(caddr, delegateABI).Read("getProfileByField", stakingAddress, "blockRewardPortion").Call(context.Background())
	if err != nil {
		log.L().Fatal("Failed to get block reward portion", zap.Error(err))
	}
	var brp []byte
	if err := blockRewardPortion.Unmarshal(&brp); err != nil {
		log.L().Fatal("Failed to unmarshal block reward portion", zap.Error(err))
	}
	if len(brp) > 0 {
		blockPortion, err := strconv.ParseInt(hex.EncodeToString(brp), 16, 64)
		if err != nil {
			log.L().Fatal("Failed to parse block reward portion", zap.Error(err))
		}
		blockRewardPercentage = float64(blockPortion) / 100
	}
	fmt.Println(blockRewardPercentage)

	// read epoch reward
	epochRewardPortion, err := c.Contract(caddr, delegateABI).Read("getProfileByField", stakingAddress, "epochRewardPortion").Call(context.Background())
	if err != nil {
		log.L().Fatal("Failed to get epoch reward portion", zap.Error(err))
	}
	var erp []byte
	if err := epochRewardPortion.Unmarshal(&erp); err != nil {
		log.L().Fatal("Failed to unmarshal epoch reward portion", zap.Error(err))
	}
	if len(erp) > 0 {
		epochPortion, err := strconv.ParseInt(hex.EncodeToString(erp), 16, 64)
		if err != nil {
			log.L().Fatal("Failed to parse epoch reward portion", zap.Error(err))
		}
		epochRewardPercentage = float64(epochPortion) / 100
	}

	// read foundation bonus
	foundationBonusPortion, err := c.Contract(caddr, delegateABI).Read("getProfileByField", stakingAddress, "foundationRewardPortion").Call(context.Background())
	if err != nil {
		log.L().Fatal("Failed to get foundation bonus portion", zap.Error(err))
	}
	var fbp []byte
	if err := foundationBonusPortion.Unmarshal(&fbp); err != nil {
		log.L().Fatal("Failed to unmarshal foundation bonus portion", zap.Error(err))
	}
	if len(fbp) > 0 {
		foundationBonusPortion, err := strconv.ParseInt(hex.EncodeToString(fbp), 16, 64)
		if err != nil {
			log.L().Fatal("Failed to parse foundation bonus portion", zap.Error(err))
		}
		foundationBonusPercentage = float64(foundationBonusPortion) / 100
	}
	return
}

func calculateVoteWeight(cfg indexprotocol.VoteWeightCalConsts, v *iotextypes.VoteBucket, selfStake bool) (*big.Int, error) {
	remainingTime := float64(v.StakedDuration * 86400)
	weight := float64(1)
	var m float64
	if v.AutoStake {
		m = cfg.AutoStake
	}
	if remainingTime > 0 {
		weight += math.Log(math.Ceil(remainingTime/86400)*(1+m)) / math.Log(cfg.DurationLg) / 100
	}
	if selfStake && v.AutoStake && v.StakedDuration >= 91 {
		// self-stake extra bonus requires enable auto-stake for at least 3 months
		weight *= cfg.SelfStake
	}

	amountInt, ok := big.NewInt(0).SetString(v.StakedAmount, 10)
	if !ok {
		return nil, errors.New("failed to convert string to big int")
	}
	amount := new(big.Float).SetInt(amountInt)
	weightedAmount, _ := amount.Mul(amount, big.NewFloat(weight)).Int(nil)
	return weightedAmount, nil
}

func remainingTime(bucket *iotextypes.VoteBucket) time.Duration {
	now := time.Now()
	startTime := time.Unix(bucket.StakeStartTime.Seconds, int64(bucket.StakeStartTime.Nanos))
	if now.Before(startTime) {
		return 0
	}
	duration := time.Duration(bucket.StakedDuration) * 24 * time.Hour
	if !bucket.AutoStake {
		endTime := startTime.Add(duration)
		if endTime.After(now) {
			return endTime.Sub(now)
		}
		return 0
	}
	return duration
}

func selfStakeIndexMap(candidates *iotextypes.CandidateListV2) map[uint64]struct{} {
	ret := make(map[uint64]struct{})
	for _, can := range candidates.Candidates {
		ret[can.SelfStakeBucketIdx] = struct{}{}
	}
	return ret
}

func ownerAddressToNameMap(candidates *iotextypes.CandidateListV2) (ret map[string]string, err error) {
	ret = make(map[string]string)
	for _, can := range candidates.Candidates {
		var name string
		name, err = indexprotocol.EncodeDelegateName(can.Name)
		if err != nil {
			return
		}
		ret[can.OwnerAddress] = name
	}
	return
}
