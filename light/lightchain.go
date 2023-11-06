/**
*  @file
*  @copyright defined in scdo/LICENSE
 */

package light

import (
	"math/big"
	"sync"

	"github.com/elcn233/go-scdo/common"
	"github.com/elcn233/go-scdo/common/errors"
	"github.com/elcn233/go-scdo/consensus"
	"github.com/elcn233/go-scdo/core"
	"github.com/elcn233/go-scdo/core/state"
	"github.com/elcn233/go-scdo/core/store"
	"github.com/elcn233/go-scdo/core/types"
	"github.com/elcn233/go-scdo/database"
	"github.com/elcn233/go-scdo/event"
	"github.com/elcn233/go-scdo/log"
)

// LightChain represents a canonical chain that by default only handles block headers.
type LightChain struct {
	mutex                     sync.RWMutex
	bcStore                   store.BlockchainStore
	odrBackend                *odrBackend
	engine                    consensus.Engine
	currentHeader             *types.BlockHeader
	canonicalTD               *big.Int
	headerChangedEventManager *event.EventManager
	headRollbackEventManager  *event.EventManager
	log                       *log.ScdoLog
}

// newLightChain create light chain
func newLightChain(bcStore store.BlockchainStore, lightDB database.Database, odrBackend *odrBackend, engine consensus.Engine) (*LightChain, error) {
	chain := &LightChain{
		bcStore:                   bcStore,
		odrBackend:                odrBackend,
		engine:                    engine,
		headerChangedEventManager: event.NewEventManager(),
		headRollbackEventManager:  event.NewEventManager(),
		log:                       log.GetLogger("LightChain"),
	}

	currentHeaderHash, err := bcStore.GetHeadBlockHash()
	if err != nil {
		return nil, errors.NewStackedError(err, "failed to get HEAD block hash")
	}

	chain.currentHeader, err = bcStore.GetBlockHeader(currentHeaderHash)
	if err != nil {
		return nil, errors.NewStackedErrorf(err, "failed to get block header, hash = %v", currentHeaderHash)
	}

	td, err := bcStore.GetBlockTotalDifficulty(currentHeaderHash)
	if err != nil {
		return nil, errors.NewStackedErrorf(err, "failed to get block TD, hash = %v", currentHeaderHash)
	}

	chain.canonicalTD = td

	return chain, nil
}

// GetState get statedb by root hash(not supported, just implement the interface here)
func (lc *LightChain) GetState(root common.Hash) (*state.Statedb, error) {
	panic("unsupported")
}

// GetStateByRootAndBlockHash get the statedb by root and block hash
func (lc *LightChain) GetStateByRootAndBlockHash(root, blockHash common.Hash) (*state.Statedb, error) {
	trie := newOdrTrie(lc.odrBackend, root, state.TrieDbPrefix, blockHash)
	return state.NewStatedbWithTrie(trie), nil
}

// CurrentHeader returns the HEAD block header of the blockchain.
func (lc *LightChain) CurrentHeader() *types.BlockHeader {
	return lc.currentHeader
}

// GetStore get underlying store
func (lc *LightChain) GetStore() store.BlockchainStore {
	return lc.bcStore
}

// GetHeader retrieves a block header from the database by height.
func (lc *LightChain) GetHeaderByHeight(height uint64) *types.BlockHeader {
	hash, err := lc.bcStore.GetBlockHash(height)
	if err != nil {
		lc.log.Warn("get block header by height failed, err %s. height %d", err, height)
		return nil
	}

	return lc.GetHeaderByHash(hash)
}

// GetHeaderByNumber retrieves a block header from the database by hash.
func (lc *LightChain) GetHeaderByHash(hash common.Hash) *types.BlockHeader {
	header, err := lc.bcStore.GetBlockHeader(hash)
	if err != nil {
		lc.log.Debug("get block header by hash failed, err %s, hash: %v", err, hash)
		return nil
	}

	return header
}

// GetHeaderByHash
func (lc *LightChain) GetBlockByHash(hash common.Hash) *types.Block {
	// this is only provided for miner interface. for light chain, there is no mining, so just return nil.
	return nil
}

// WriteHeader writes the specified block header to the blockchain.
func (lc *LightChain) WriteHeader(header *types.BlockHeader) error {
	lc.mutex.Lock()
	defer lc.mutex.Unlock()

	if err := core.ValidateBlockHeader(header, lc.engine, lc.bcStore, lc); err != nil {
		return errors.NewStackedError(err, "failed to validate block header")
	}

	previousTd, err := lc.bcStore.GetBlockTotalDifficulty(header.PreviousBlockHash)
	if err != nil {
		return errors.NewStackedErrorf(err, "failed to get block TD, hash = %v", header.PreviousBlockHash)
	}

	currentTd := new(big.Int).Add(previousTd, header.Difficulty)
	isHead := currentTd.Cmp(lc.canonicalTD) > 0

	if err := lc.bcStore.PutBlockHeader(header.Hash(), header, currentTd, isHead); err != nil {
		return errors.NewStackedErrorf(err, "failed to put block header, header = %+v", header)
	}

	if !isHead {
		return nil
	}

	if err := core.DeleteLargerHeightBlocks(lc.bcStore, header.Height+1, nil); err != nil {
		return errors.NewStackedErrorf(err, "failed to delete larger height blocks in canonical chain, height = %v", header.Height+1)
	}

	if err := core.OverwriteStaleBlocks(lc.bcStore, header.PreviousBlockHash, nil); err != nil {
		return errors.NewStackedErrorf(err, "failed to overwrite stale blocks in old canonical chain, hash = %v", header.PreviousBlockHash)
	}

	lc.canonicalTD = currentTd
	lc.currentHeader = header

	lc.headerChangedEventManager.Fire(header)

	return nil
}

// GetCurrentState get current state
func (lc *LightChain) GetCurrentState() (*state.Statedb, error) {
	return lc.GetStateByRootAndBlockHash(lc.currentHeader.StateHash, lc.currentHeader.Hash())
}

// GetHeadRollbackEventManager
func (lc *LightChain) GetHeadRollbackEventManager() *event.EventManager {
	return lc.headRollbackEventManager
}

// PutTd set light chain canonial total difficulty
func (lc *LightChain) PutTd(td *big.Int) {
	lc.canonicalTD = td
}

// PutCurrentHeader
func (lc *LightChain) PutCurrentHeader(header *types.BlockHeader) {
	lc.currentHeader = header
}
