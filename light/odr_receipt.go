/**
*  @file
*  @copyright defined in scdo/LICENSE
 */

package light

import (
	"github.com/elcn233/go-scdo/api"
	"github.com/elcn233/go-scdo/common"
	"github.com/elcn233/go-scdo/common/errors"
	"github.com/elcn233/go-scdo/core/store"
	"github.com/elcn233/go-scdo/core/types"
)

var errReceipIndexNil = errors.New("got a nil receipt index")

type odrReceiptRequest struct {
	OdrItem
	TxHash common.Hash
}

type odrReceiptResponse struct {
	OdrProvableResponse
	Receipt *types.Receipt `rlp:"nil"`
}

func (request *odrReceiptRequest) code() uint16 {
	return receiptRequestCode
}

func (request *odrReceiptRequest) handle(lp *LightProtocol) (uint16, odrResponse) {
	txIndex, err := lp.chain.GetStore().GetTxIndex(request.TxHash)
	if err != nil {
		err = errors.NewStackedErrorf(err, "failed to get tx index by hash %v", request.TxHash)
		return newErrorResponse(receiptResponseCode, request.ReqID, err)
	}

	header, err := lp.chain.GetStore().GetBlockHeader(txIndex.BlockHash)
	if err != nil {
		err = errors.NewStackedErrorf(err, "failed to get block header by hash %v", txIndex.BlockHash)
		return newErrorResponse(receiptResponseCode, request.ReqID, err)
	}

	receipts, err := lp.chain.GetStore().GetReceiptsByBlockHash(txIndex.BlockHash)
	if err != nil {
		err = errors.NewStackedErrorf(err, "failed to get receipts by block hash %v", txIndex.BlockHash)
		return newErrorResponse(receiptResponseCode, request.ReqID, err)
	}

	var result odrReceiptResponse
	result.ReqID = request.ReqID
	result.Receipt = receipts[txIndex.Index]
	result.BlockIndex = &api.BlockIndex{
		BlockHash:   txIndex.BlockHash,
		BlockHeight: header.Height,
		Index:       txIndex.Index,
	}

	receiptTrie := types.GetReceiptTrie(receipts)
	proof, err := receiptTrie.GetProof(result.Receipt.TxHash.Bytes())
	if err != nil {
		err = errors.NewStackedError(err, "failed to get receipt trie proof")
		return newErrorResponse(receiptResponseCode, request.ReqID, err)
	}

	result.Proof = mapToArray(proof)

	return receiptResponseCode, &result
}

func (response *odrReceiptResponse) validate(request odrRequest, bcStore store.BlockchainStore) error {
	header, err := response.proveHeader(bcStore)
	if err != nil {
		return errors.NewStackedError(err, "failed to prove block header")
	}

	if header == nil {
		return errReceipIndexNil
	}

	txHash := request.(*odrReceiptRequest).TxHash
	response.Receipt = new(types.Receipt)
	if err := response.proveMerkleTrie(header.ReceiptHash, txHash.Bytes(), response.Receipt); err != nil {
		return errors.NewStackedError(err, "failed to prove merkle trie")
	}

	return nil
}
