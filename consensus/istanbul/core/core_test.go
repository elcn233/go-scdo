/**
*  @file
*  @copyright defined in scdo/LICENSE
 */

package core

import (
	"math/big"
	"reflect"
	"testing"
	"time"

	"github.com/elcn233/go-scdo/consensus/istanbul"
	"github.com/elcn233/go-scdo/core/types"
)

func makeBlock(number int64) *types.Block {
	header := &types.BlockHeader{
		Difficulty:      big.NewInt(0),
		Height:          uint64(number),
		CreateTimestamp: big.NewInt(0),
	}
	block := &types.Block{}
	return block.WithSeal(header)
}

func newTestProposal() istanbul.Proposal {
	return makeBlock(1)
}

func TestNewRequest(t *testing.T) {
	N := uint64(4)
	F := uint64(1)

	sys := NewTestSystemWithBackend(N, F)

	close := sys.Run(true)
	defer close()

	request1 := makeBlock(1)
	sys.backends[0].NewRequest(request1)

	select {
	case <-time.After(1 * time.Second):
	}

	request2 := makeBlock(2)
	sys.backends[0].NewRequest(request2)

	select {
	case <-time.After(1 * time.Second):
	}

	for _, backend := range sys.backends {
		if len(backend.committedMsgs) != 2 {
			t.Errorf("the number of executed requests mismatch: have %v, want 2", len(backend.committedMsgs))
		}
		if !reflect.DeepEqual(request1.Height(), backend.committedMsgs[0].commitProposal.Height()) {
			t.Errorf("the number of requests mismatch: have %v, want %v", request1.Height(), backend.committedMsgs[0].commitProposal.Height())
		}
		if !reflect.DeepEqual(request2.Height(), backend.committedMsgs[1].commitProposal.Height()) {
			t.Errorf("the number of requests mismatch: have %v, want %v", request2.Height(), backend.committedMsgs[1].commitProposal.Height())
		}
	}
}
