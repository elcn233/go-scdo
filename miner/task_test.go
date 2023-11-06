/**
*  @file
*  @copyright defined in scdo/LICENSE
 */

package miner

import (
	"math/big"
	"sync"
	"testing"
	"time"

	"github.com/elcn233/go-scdo/common"
	"github.com/elcn233/go-scdo/consensus"
	"github.com/elcn233/go-scdo/consensus/pow"
	"github.com/elcn233/go-scdo/core"
	"github.com/elcn233/go-scdo/core/state"
	"github.com/elcn233/go-scdo/core/types"
	"github.com/elcn233/go-scdo/crypto"
	"github.com/elcn233/go-scdo/database/leveldb"
	"github.com/elcn233/go-scdo/log"
	"github.com/stretchr/testify/assert"
)

func getTask(difficult int64) *Task {
	return &Task{
		header: &types.BlockHeader{
			Difficulty: big.NewInt(difficult),
		},
	}
}

func newTestBlockHeader() *types.BlockHeader {
	return &types.BlockHeader{
		PreviousBlockHash: common.StringToHash("PreviousBlockHash"),
		Creator:           common.BytesToAddress([]byte{1}),
		StateHash:         common.StringToHash("StateHash"),
		TxHash:            common.StringToHash("TxHash"),
		Difficulty:        big.NewInt(1),
		Height:            1,
		CreateTimestamp:   big.NewInt(time.Now().Unix()),
		Witness:           make([]byte, 0),
		ExtraData:         common.CopyBytes([]byte("ExtraData")),
	}
}

func Test_handleMinerRewardTx(t *testing.T) {
	db, remove := leveldb.NewTestDatabase()
	defer remove()

	statedb, err := state.NewStatedb(common.EmptyHash, db)
	if err != nil {
		panic(err)
	}

	task := getTask(10)
	task.header = newTestBlockHeader()
	reward, err := task.handleMinerRewardTx(statedb)

	assert.Equal(t, err, nil)
	assert.Equal(t, reward, consensus.GetReward(task.header.Height))
}

func Test_ChooseTransactionAndDebts(t *testing.T) {
	verifier := types.NewTestVerifier(true, false, nil)
	task, debtPool := testWithBackend(verifier, t)

	assert.Equal(t, 6, len(task.Transactions))
	assert.Equal(t, 0, len(task.Debts))
	assert.Equal(t, 3, debtPool.GetDebtCount(false, true))

	verifier2 := types.NewTestVerifier(true, true, nil)
	task, debtPool = testWithBackend(verifier2, t)

	assert.Equal(t, 6, len(task.Transactions))
	assert.Equal(t, 3, len(task.Debts))
	assert.Equal(t, 0, debtPool.GetDebtCount(false, true))
	assert.Equal(t, 3, debtPool.GetDebtCount(true, false))
}

func testWithBackend(verifier types.DebtVerifier, t *testing.T) (*types.Block, *core.DebtPool) {
	backend := NewTestScdoBackendWithVerifier(verifier)

	bc := backend.BlockChain()
	parent := bc.Genesis()
	coinbase := *crypto.MustGenerateShardAddress(types.TestGenesisShard)
	header := newHeaderByParent(parent, coinbase, time.Now().Unix())
	task := NewTask(header, coinbase, verifier)

	engine := pow.NewEngine(1)
	engine.Prepare(bc, header)

	txPool := backend.TxPool()
	txPool.AddTransaction(types.NewTestTransactionWithNonce(0))
	txPool.AddTransaction(types.NewTestTransactionWithNonce(1))
	txPool.AddTransaction(types.NewTestTransactionWithNonce(2))
	txPool.AddTransaction(types.NewTestCrossShardTransactionWithNonce(3))
	txPool.AddTransaction(types.NewTestCrossShardTransactionWithNonce(4))

	debtPool := backend.DebtPool()
	debtPool.AddDebt(types.NewTestDebtWithTargetShard(1))
	debtPool.AddDebt(types.NewTestDebtWithTargetShard(1))
	debtPool.AddDebt(types.NewTestDebtWithTargetShard(1))
	debtPool.DoCheckingDebt()

	state, err := state.NewStatedb(parent.Header.StateHash, bc.AccountDB())
	assert.Equal(t, err, nil)

	log := log.GetLogger("test_task")
	err = task.applyTransactionsAndDebts(backend, state, log)
	assert.Equal(t, err, nil)

	block := task.generateBlock()
	result := make(chan *types.Block)
	var resultBlock *types.Block
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		resultBlock = <-result
	}()

	err = engine.Seal(bc, block, make(chan struct{}), result)
	assert.Equal(t, nil, err)

	wg.Wait()

	err = bc.WriteBlock(resultBlock)
	assert.Equal(t, nil, err)

	return resultBlock, debtPool
}
