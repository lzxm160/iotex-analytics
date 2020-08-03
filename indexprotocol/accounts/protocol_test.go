package accounts

import (
	"context"
	"database/sql"
	"testing"

	"github.com/iotexproject/iotex-analytics/indexcontext"
	"github.com/iotexproject/iotex-core/test/mock/mock_apiserviceclient"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"

	"github.com/iotexproject/iotex-analytics/epochctx"
	s "github.com/iotexproject/iotex-analytics/sql"
	"github.com/iotexproject/iotex-analytics/testutil"
)

const (
	connectStr = "be10c04ac183b5:0a8f49f9@tcp(us-cdbr-east-02.cleardb.com:3306)/"
	dbName     = "heroku_88b589bc76fadbc"
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

	p := NewProtocol(store, epochctx.NewEpochCtx(1, 1, 1))

	require.NoError(p.CreateTables(ctx))

	blk, err := testutil.BuildCompleteBlock(uint64(1), uint64(2))
	require.NoError(err)
	chainClient := mock_apiserviceclient.NewMockServiceClient(ctrl)
	ctx = indexcontext.WithIndexCtx(context.Background(), indexcontext.IndexCtx{
		ChainClient:     chainClient,
		ConsensusScheme: "ROLLDPOS",
	})
	chainClient.EXPECT().GetTransactionLogByBlockHeight(gomock.Any(), gomock.Any()).Times(1).Return(nil, nil)

	require.NoError(store.Transact(func(tx *sql.Tx) error {
		return p.HandleBlock(ctx, tx, blk)
	}))

	blk2, err := testutil.BuildEmptyBlock(2)
	require.NoError(err)

	require.NoError(store.Transact(func(tx *sql.Tx) error {
		return p.HandleBlock(ctx, tx, blk2)
	}))

	// get balance history
	balanceHistory, err := p.getBalanceHistory(testutil.Addr1)
	require.NoError(err)
	require.Equal(2, len(balanceHistory))
	require.Equal("2", balanceHistory[1].Amount)

	// get account income
	accountIncome, err := p.getAccountIncome(uint64(1), testutil.Addr1)
	require.NoError(err)
	require.Equal("-2", accountIncome.Income)
}
