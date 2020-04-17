package hermes2

import (
	"fmt"
	"strconv"

	"github.com/pkg/errors"

	"github.com/iotexproject/iotex-analytics/indexprotocol"
	"github.com/iotexproject/iotex-analytics/indexprotocol/accounts"
	"github.com/iotexproject/iotex-analytics/indexprotocol/actions"
	"github.com/iotexproject/iotex-analytics/indexservice"
	s "github.com/iotexproject/iotex-analytics/sql"
)

const (
	// SelectCountByDelegateName selects the count of Hermes distribution by delegate name
	SelectCountByDelegateName = selectCount + fromJoinedTables + delegateFilter
	// SelectCountByVoterAddress selects the count of Hermes distribution by voter address
	SelectCountByVoterAddress = selectCount + fromJoinedTables + voterFilter

	fromJoinedTables = "FROM (SELECT * FROM %s WHERE epoch_number >= ? AND epoch_number <= ? AND `from` = ?) " +
		"AS t1 INNER JOIN (SELECT * FROM %s WHERE epoch_number >= ? AND epoch_number <= ?) AS t2 ON t1.action_hash = t2.action_hash "
	timeOrdering = "ORDER BY `timestamp` desc limit ?,?"

	selectVoter                            = "SELECT `to`, from_epoch, to_epoch, amount, t1.action_hash, `timestamp` "
	delegateFilter                         = "WHERE delegate_name = ? "
	selectHermesDistributionByDelegateName = selectVoter + fromJoinedTables + delegateFilter + timeOrdering

	selectDelegate                         = "SELECT delegate_name, from_epoch, to_epoch, amount, t1.action_hash, `timestamp` "
	voterFilter                            = "WHERE `to` = ? "
	selectHermesDistributionByVoterAddress = selectDelegate + fromJoinedTables + voterFilter + timeOrdering

	selectCount = "SELECT COUNT(*),SUM(amount) "

	selectNumberOfDelegates       = "SELECT COUNT(DISTINCT delegate_name) FROM %s WHERE epoch_number >= %d AND epoch_number <= %d"
	selectNumberOfRecipients      = "SELECT COUNT(DISTINCT `to`) FROM %s WHERE `from` = '%s' and `to`<>'' AND epoch_number >= %d AND epoch_number <= %d"
	selectTotalRewardsDistributed = "SELECT SUM(amount) FROM (SELECT * FROM %s WHERE epoch_number >= %d AND epoch_number <= %d AND `from` = '%s') AS t1 INNER JOIN (SELECT * FROM %s WHERE epoch_number >= %d AND epoch_number <= %d) AS t2 ON t1.action_hash = t2.action_hash"
)

// HermesArg defines Hermes request parameters
type HermesArg struct {
	StartEpoch int
	EpochCount int
	Offset     uint64
	Size       uint64
}

// VoterInfo defines voter information
type VoterInfo struct {
	VoterAddress string
	FromEpoch    uint64
	ToEpoch      uint64
	Amount       string
	ActionHash   string
	Timestamp    string
}

// DelegateInfo defines delegate information
type DelegateInfo struct {
	DelegateName string
	FromEpoch    uint64
	ToEpoch      uint64
	Amount       string
	ActionHash   string
	Timestamp    string
}

// Protocol defines the protocol of querying tables
type Protocol struct {
	indexer      *indexservice.Indexer
	hermesConfig indexprotocol.HermesConfig
}

// NewProtocol creates a new protocol
func NewProtocol(idx *indexservice.Indexer, cfg indexprotocol.HermesConfig) *Protocol {
	return &Protocol{
		indexer:      idx,
		hermesConfig: cfg,
	}
}

// GetHermes2ByDelegate gets Hermes voter list by delegate name
func (p *Protocol) GetHermes2ByDelegate(arg HermesArg, delegateName string) ([]*VoterInfo, error) {
	db := p.indexer.Store.GetDB()
	getQuery := fmt.Sprintf(selectHermesDistributionByDelegateName, accounts.BalanceHistoryTableName, actions.HermesContractTableName)
	stmt, err := db.Prepare(getQuery)
	if err != nil {
		return nil, errors.Wrap(err, "failed to prepare get query")
	}
	defer stmt.Close()

	endEpoch := arg.StartEpoch + arg.EpochCount - 1
	rows, err := stmt.Query(arg.StartEpoch, endEpoch, p.hermesConfig.MultiSendContractAddress, arg.StartEpoch, endEpoch,
		delegateName, arg.Offset, arg.Size)
	if err != nil {
		return nil, errors.Wrap(err, "failed to execute get query")
	}

	fmt.Println(getQuery)
	fmt.Println(arg.StartEpoch, endEpoch, p.hermesConfig.MultiSendContractAddress, arg.StartEpoch, endEpoch,
		delegateName, arg.Offset, arg.Size)
	var voterInfo VoterInfo
	parsedRows, err := s.ParseSQLRows(rows, &voterInfo)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse results")
	}
	if len(parsedRows) == 0 {
		return nil, indexprotocol.ErrNotExist
	}

	voterInfoList := make([]*VoterInfo, 0)
	for _, parsedRow := range parsedRows {
		voterInfoList = append(voterInfoList, parsedRow.(*VoterInfo))
	}

	return voterInfoList, nil
}

// GetHermes2ByVoter gets Hermes delegate list by voter name
func (p *Protocol) GetHermes2ByVoter(arg HermesArg, voterAddress string) ([]*DelegateInfo, error) {
	db := p.indexer.Store.GetDB()
	getQuery := fmt.Sprintf(selectHermesDistributionByVoterAddress, accounts.BalanceHistoryTableName, actions.HermesContractTableName)
	stmt, err := db.Prepare(getQuery)
	if err != nil {
		return nil, errors.Wrap(err, "failed to prepare get query")
	}
	defer stmt.Close()

	endEpoch := arg.StartEpoch + arg.EpochCount - 1
	rows, err := stmt.Query(arg.StartEpoch, endEpoch, p.hermesConfig.MultiSendContractAddress, arg.StartEpoch, endEpoch,
		voterAddress, arg.Offset, arg.Size)
	if err != nil {
		return nil, errors.Wrap(err, "failed to execute get query")
	}
	fmt.Println(getQuery)
	fmt.Println(arg.StartEpoch, endEpoch, p.hermesConfig.MultiSendContractAddress, arg.StartEpoch, endEpoch,
		voterAddress, arg.Offset, arg.Size)
	var delegateInfo DelegateInfo
	parsedRows, err := s.ParseSQLRows(rows, &delegateInfo)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse results")
	}
	if len(parsedRows) == 0 {
		return nil, indexprotocol.ErrNotExist
	}

	delegateInfoList := make([]*DelegateInfo, 0)
	for _, parsedRow := range parsedRows {
		delegateInfoList = append(delegateInfoList, parsedRow.(*DelegateInfo))
	}

	return delegateInfoList, nil
}

// GetHermes2Count gets the count of Hermes distributions
func (p *Protocol) GetHermes2Count(arg HermesArg, selectQuery string, filter string) (count int, total string, err error) {
	db := p.indexer.Store.GetDB()
	getQuery := fmt.Sprintf(selectQuery, accounts.BalanceHistoryTableName, actions.HermesContractTableName)
	stmt, err := db.Prepare(getQuery)
	if err != nil {
		err = errors.Wrap(err, "failed to prepare get query")
		return
	}
	defer stmt.Close()

	endEpoch := arg.StartEpoch + arg.EpochCount - 1
	fmt.Println(getQuery)
	fmt.Println(arg.StartEpoch, endEpoch, p.hermesConfig.MultiSendContractAddress, arg.StartEpoch, endEpoch,
		filter)
	if err = stmt.QueryRow(arg.StartEpoch, endEpoch, p.hermesConfig.MultiSendContractAddress, arg.StartEpoch, endEpoch,
		filter).Scan(&count, &total); err != nil {
		err = errors.Wrap(err, "failed to execute get query")
		return
	}
	return
}

// GetHermes2Meta gets the hermes meta info
func (p *Protocol) GetHermes2Meta(startEpoch int, epochCount int) (numberOfDelegates int,
	numberOfRecipients int, totalRewardsDistributed string, err error) {
	endEpoch := startEpoch + epochCount - 1
	numberOfDelegates, err = p.getNumOfDelegates(startEpoch, endEpoch)
	if err != nil {
		err = errors.Wrap(err, "get num of delegates")
		return
	}
	numberOfRecipients, err = p.getNumOfReceipts(startEpoch, endEpoch)
	if err != nil {
		err = errors.Wrap(err, "get num of receipts")
		return
	}
	totalRewardsDistributed, err = p.getTotalRewardsDistributed(startEpoch, endEpoch)
	if err != nil {
		err = errors.Wrap(err, "get num of total rewards distributed")
		return
	}
	return
}

func (p *Protocol) getNumOfDelegates(startEpoch int, endEpoch int) (numberOfDelegates int, err error) {
	db := p.indexer.Store.GetDB()
	getQuery := fmt.Sprintf(selectNumberOfDelegates, actions.HermesContractTableName, startEpoch, endEpoch)
	fmt.Println(getQuery)
	num, err := s.GetCount(db, getQuery)
	if num == "0" {
		err = indexprotocol.ErrNotExist
		return
	}
	numberOfDelegates, err = strconv.Atoi(num)
	if err != nil {
		err = errors.Wrap(err, "str conv:"+num)
		return
	}
	return
}

func (p *Protocol) getNumOfReceipts(startEpoch int, endEpoch int) (numberOfeceipts int, err error) {
	db := p.indexer.Store.GetDB()
	getQuery := fmt.Sprintf(selectNumberOfRecipients, accounts.BalanceHistoryTableName, p.hermesConfig.MultiSendContractAddress, startEpoch, endEpoch)
	fmt.Println(getQuery)
	num, err := s.GetCount(db, getQuery)
	if num == "0" {
		err = indexprotocol.ErrNotExist
		return
	}
	numberOfeceipts, err = strconv.Atoi(num)
	if err != nil {
		err = errors.Wrap(err, "str conv:"+num)
		return
	}
	return
}

func (p *Protocol) getTotalRewardsDistributed(startEpoch int, endEpoch int) (totalRewardsDistributed string, err error) {
	db := p.indexer.Store.GetDB()
	getQuery := fmt.Sprintf(selectTotalRewardsDistributed, accounts.BalanceHistoryTableName, startEpoch, endEpoch, p.hermesConfig.MultiSendContractAddress, actions.HermesContractTableName, startEpoch, endEpoch)
	fmt.Println(getQuery)
	totalRewardsDistributed, err = s.GetCount(db, getQuery)
	if totalRewardsDistributed == "0" {
		err = indexprotocol.ErrNotExist
		return
	}
	return
}
