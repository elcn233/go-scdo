/**
*  @file
*  @copyright defined in scdo/LICENSE
 */

package light

import (
	"math/big"
	"strings"
	"testing"

	"github.com/elcn233/go-scdo/common"
	"github.com/elcn233/go-scdo/common/errors"
	"github.com/elcn233/go-scdo/consensus"
	"github.com/elcn233/go-scdo/consensus/pow"
	"github.com/elcn233/go-scdo/core/store"
	"github.com/elcn233/go-scdo/core/types"
	"github.com/elcn233/go-scdo/crypto"
	"github.com/elcn233/go-scdo/database"
	"github.com/elcn233/go-scdo/database/leveldb"
	"github.com/stretchr/testify/assert"
)

func newTestBlockchainDatabase(db database.Database) store.BlockchainStore {
	return store.NewBlockchainDatabase(db)
}

func newTestLightChain() (*LightChain, func(), error) {
	db, dispose := leveldb.NewTestDatabase()
	bcStore := newTestBlockchainDatabase(db)
	backend := newOdrBackend(bcStore, 1)

	// put genesis block
	header := newTestBlockHeader()
	headerHash := header.Hash()
	bcStore.PutBlockHeader(headerHash, header, header.Difficulty, true)

	lc, err := newLightChain(bcStore, db, backend, pow.NewEngine(1))
	return lc, dispose, err
}

func newTestBlockHeader() *types.BlockHeader {
	return &types.BlockHeader{
		PreviousBlockHash: common.EmptyHash,
		Creator:           common.HexMustToAddres("0x55c76ac9f0d4de0efb11207cb67cf13f01357fc1"),
		StateHash:         common.StringToHash("StateHash"),
		TxHash:            common.StringToHash("TxHash"),
		Difficulty:        big.NewInt(1),
		Height:            1,
		CreateTimestamp:   big.NewInt(1),
		Witness:           make([]byte, 0),
		ExtraData:         make([]byte, 0),
	}
}

func newTestNonGensisBlockHeader(parentHeader *types.BlockHeader, difficulty *big.Int, height uint64) *types.BlockHeader {
	return &types.BlockHeader{
		PreviousBlockHash: parentHeader.Hash(),
		Creator:           *crypto.MustGenerateRandomAddress(),
		StateHash:         common.StringToHash("StateHash"),
		TxHash:            common.StringToHash("TxHash"),
		Difficulty:        difficulty,
		Height:            height,
		CreateTimestamp:   big.NewInt(2),
		Witness:           make([]byte, 0),
		ExtraData:         make([]byte, 0),
	}
}

func Test_LightChain_NewLightChain(t *testing.T) {
	db, dispose := leveldb.NewTestDatabase()
	defer dispose()

	bcStore := newTestBlockchainDatabase(db)
	backend := newOdrBackend(bcStore, 1)

	// no block in bcStore
	lc, err := newLightChain(bcStore, db, backend, pow.NewEngine(1))
	assert.Equal(t, strings.Contains(err.Error(), "leveldb: not found"), true)
	assert.Equal(t, lc == nil, true)

	// put genesis block
	header := newTestBlockHeader()
	headerHash := header.Hash()
	bcStore.PutBlockHeader(headerHash, header, header.Difficulty, true)

	lc, err = newLightChain(bcStore, db, backend, pow.NewEngine(1))
	assert.Equal(t, err, nil)
	assert.Equal(t, lc != nil, true)
	assert.Equal(t, lc.currentHeader != nil, true)
	assert.Equal(t, lc.canonicalTD, big.NewInt(1))
}

func Test_LightChain_GetState(t *testing.T) {
	lc, dispose, _ := newTestLightChain()
	defer dispose()

	state, err := lc.GetStateByRootAndBlockHash(common.EmptyHash, common.EmptyHash)
	assert.Equal(t, err, nil)
	assert.Equal(t, state != nil, true)

	state, err = lc.GetCurrentState()
	assert.Equal(t, err, nil)
	assert.Equal(t, state != nil, true)
}

func Test_LightChain_WriteHeader(t *testing.T) {
	lc, dispose, _ := newTestLightChain()
	defer dispose()

	blockHeader := newTestNonGensisBlockHeader(newTestBlockHeader(), big.NewInt(1), 1)
	err := lc.WriteHeader(blockHeader)
	assert.True(t, errors.IsOrContains(err, consensus.ErrBlockInvalidHeight))

	blockHeader = newTestNonGensisBlockHeader(newTestBlockHeader(), big.NewInt(1), 2)
	err = lc.WriteHeader(blockHeader)
	assert.Equal(t, err, nil)
}
