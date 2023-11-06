/**
*  @file
*  @copyright defined in scdo/LICENSE
 */

package light

import (
	"github.com/elcn233/go-scdo/common"
	"github.com/elcn233/go-scdo/common/errors"
	"github.com/elcn233/go-scdo/core/store"
	"github.com/elcn233/go-scdo/core/types"
)

type odrBlock struct {
	OdrItem
	Hash  common.Hash  // Retrieved block hash
	Block *types.Block `rlp:"nil"` // Retrieved block
}

func (odr *odrBlock) code() uint16 {
	return blockRequestCode
}

// handle handle odr request
func (odr *odrBlock) handle(lp *LightProtocol) (uint16, odrResponse) {
	var err error

	if odr.Block, err = lp.chain.GetStore().GetBlock(odr.Hash); err != nil {
		lp.log.Debug("Failed to get block, hash = %v, error = %v", odr.Hash, err)
		odr.Error = errors.NewStackedErrorf(err, "failed to get block by hash %v", odr.Hash).Error()
	}

	return blockResponseCode, odr
}

// validate validate request info against local store
func (odr *odrBlock) validate(request odrRequest, bcStore store.BlockchainStore) error {
	if odr.Block == nil {
		return nil
	}

	if err := odr.Block.Validate(); err != nil {
		return errors.NewStackedError(err, "failed to validate block")
	}

	if hash := request.(*odrBlock).Hash; !hash.Equal(odr.Block.HeaderHash) {
		return types.ErrBlockHashMismatch
	}

	return nil
}
