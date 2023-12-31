/**
* @file
* @copyright defined in scdo/LICENSE
 */

package types

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/elcn233/go-scdo/common"
	"github.com/elcn233/go-scdo/crypto"
	"github.com/stretchr/testify/assert"
)

func Test_NewDebt(t *testing.T) {
	tx1 := newTestTxWithSign(1, 1, 1, true)

	d1 := NewDebtWithContext(tx1)
	assert.Equal(t, d1.Data.Amount, big.NewInt(1))
	assert.Equal(t, d1.Data.Account, tx1.Data.To)
	assert.Equal(t, d1.Data.From.Shard(), tx1.Data.From.Shard())
	assert.Equal(t, d1.Data.TxHash, tx1.Hash)
	assert.Equal(t, d1.Hash, crypto.MustHash(d1.Data))
}

func Test_MerkleRoot(t *testing.T) {
	debts := make([]*Debt, 0)

	for i := 0; i < 100; i++ {
		tx := newTestTxWithSign(1, 1, 1, true)
		d := NewDebtWithContext(tx)

		debts = append(debts, d)
	}

	common.LocalShardNumber = 1
	defer func() {
		common.LocalShardNumber = common.UndefinedShardNumber
	}()

	hash := DebtMerkleRootHash(debts)
	if hash == common.EmptyHash {
		t.Fatal("got empty hash")
	}
}

func Test_DebtSize(t *testing.T) {
	tx := newTestTxWithSign(1, 1, 1, true)

	d := NewDebtWithContext(tx)

	array := []*Debt{d}
	buff := common.SerializePanic(array)
	fmt.Println(len(buff))
	assert.Equal(t, len(buff), DebtSize)

	array = []*Debt{d, d}
	buff = common.SerializePanic(array)
	fmt.Println(len(buff) / 2)
	assert.Equal(t, len(buff)/2, DebtSize-1)

	array = []*Debt{d, d, d}
	buff = common.SerializePanic(array)
	fmt.Println(len(buff) / 3)
	assert.Equal(t, len(buff)/3, DebtSize-1)

	array = []*Debt{d, d, d, d}
	buff = common.SerializePanic(array)
	fmt.Println(len(buff) / 4)
	assert.Equal(t, len(buff)/4, DebtSize-2)

	array = []*Debt{d, d, d, d, d}
	buff = common.SerializePanic(array)
	fmt.Println(len(buff) / 5)
	assert.Equal(t, len(buff)/5, DebtSize-2)
}
