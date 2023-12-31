/**
*  @file
*  @copyright defined in scdo/LICENSE
 */

package backend

import (
	"bytes"
	"crypto/ecdsa"
	"math/big"
	"reflect"
	"testing"

	"github.com/elcn233/go-scdo/common"
	"github.com/elcn233/go-scdo/consensus/istanbul"
	"github.com/elcn233/go-scdo/consensus/istanbul/validator"
	core2 "github.com/elcn233/go-scdo/core"
	"github.com/elcn233/go-scdo/core/store"
	"github.com/elcn233/go-scdo/core/types"
	"github.com/elcn233/go-scdo/crypto"
	"github.com/elcn233/go-scdo/database/leveldb"
)

type testerVote struct {
	validator string
	voted     string
	auth      bool
}

// testerAccountPool is a pool to maintain currently active tester accounts,
// mapped from textual names used in the tests below to actual Ethereum private
// keys capable of signing transactions.
type testerAccountPool struct {
	accounts map[string]*ecdsa.PrivateKey
}

func newTesterAccountPool() *testerAccountPool {
	return &testerAccountPool{
		accounts: make(map[string]*ecdsa.PrivateKey),
	}
}

func (ap *testerAccountPool) sign(header *types.BlockHeader, validator string) {
	// Ensure we have a persistent key for the validator
	if ap.accounts[validator] == nil {
		ap.accounts[validator], _ = crypto.GenerateKey()
	}
	// Sign the header and embed the signature in extra data
	hashData := crypto.Keccak256(sigHash(header).Bytes())
	sig, _ := crypto.Sign(ap.accounts[validator], hashData)

	writeSeal(header, sig.Sig)
}

func (ap *testerAccountPool) address(account string) common.Address {
	// Ensure we have a persistent key for the account
	if ap.accounts[account] == nil {
		ap.accounts[account], _ = crypto.GenerateKey()
	}
	// Resolve and return the Ethereum address
	return crypto.PubkeyToAddress(ap.accounts[account].PublicKey)
}

// Tests that voting is evaluated correctly for various simple and complex scenarios.
func TestVoting(t *testing.T) {
	// Define the various voting scenarios to test
	tests := []struct {
		epoch      uint64
		validators []string
		votes      []testerVote
		results    []string
	}{
		{
			// Single validator, no votes cast
			validators: []string{"A"},
			votes:      []testerVote{{validator: "A"}},
			results:    []string{"A"},
		}, {
			// Single validator, voting to add two others (only accept first, second needs 2 votes)
			validators: []string{"A"},
			votes: []testerVote{
				{validator: "A", voted: "B", auth: true},
				{validator: "B"},
				{validator: "A", voted: "C", auth: true},
			},
			results: []string{"A", "B"},
		}, {
			// Two validators, voting to add three others (only accept first two, third needs 3 votes already)
			validators: []string{"A", "B"},
			votes: []testerVote{
				{validator: "A", voted: "C", auth: true},
				{validator: "B", voted: "C", auth: true},
				{validator: "A", voted: "D", auth: true},
				{validator: "B", voted: "D", auth: true},
				{validator: "C"},
				{validator: "A", voted: "E", auth: true},
				{validator: "B", voted: "E", auth: true},
			},
			results: []string{"A", "B", "C", "D"},
		}, {
			// Single validator, dropping itself (weird, but one less cornercase by explicitly allowing this)
			validators: []string{"A"},
			votes: []testerVote{
				{validator: "A", voted: "A", auth: false},
			},
			results: []string{},
		}, {
			// Two validators, actually needing mutual consent to drop either of them (not fulfilled)
			validators: []string{"A", "B"},
			votes: []testerVote{
				{validator: "A", voted: "B", auth: false},
			},
			results: []string{"A", "B"},
		}, {
			// Two validators, actually needing mutual consent to drop either of them (fulfilled)
			validators: []string{"A", "B"},
			votes: []testerVote{
				{validator: "A", voted: "B", auth: false},
				{validator: "B", voted: "B", auth: false},
			},
			results: []string{"A"},
		}, {
			// Three validators, two of them deciding to drop the third
			validators: []string{"A", "B", "C"},
			votes: []testerVote{
				{validator: "A", voted: "C", auth: false},
				{validator: "B", voted: "C", auth: false},
			},
			results: []string{"A", "B"},
		}, {
			// Four validators, consensus of two not being enough to drop anyone
			validators: []string{"A", "B", "C", "D"},
			votes: []testerVote{
				{validator: "A", voted: "C", auth: false},
				{validator: "B", voted: "C", auth: false},
			},
			results: []string{"A", "B", "C", "D"},
		}, {
			// Four validators, consensus of three already being enough to drop someone
			validators: []string{"A", "B", "C", "D"},
			votes: []testerVote{
				{validator: "A", voted: "D", auth: false},
				{validator: "B", voted: "D", auth: false},
				{validator: "C", voted: "D", auth: false},
			},
			results: []string{"A", "B", "C"},
		}, {
			// Authorizations are counted once per validator per target
			validators: []string{"A", "B"},
			votes: []testerVote{
				{validator: "A", voted: "C", auth: true},
				{validator: "B"},
				{validator: "A", voted: "C", auth: true},
				{validator: "B"},
				{validator: "A", voted: "C", auth: true},
			},
			results: []string{"A", "B"},
		}, {
			// Authorizing multiple accounts concurrently is permitted
			validators: []string{"A", "B"},
			votes: []testerVote{
				{validator: "A", voted: "C", auth: true},
				{validator: "B"},
				{validator: "A", voted: "D", auth: true},
				{validator: "B"},
				{validator: "A"},
				{validator: "B", voted: "D", auth: true},
				{validator: "A"},
				{validator: "B", voted: "C", auth: true},
			},
			results: []string{"A", "B", "C", "D"},
		}, {
			// Deauthorizations are counted once per validator per target
			validators: []string{"A", "B"},
			votes: []testerVote{
				{validator: "A", voted: "B", auth: false},
				{validator: "B"},
				{validator: "A", voted: "B", auth: false},
				{validator: "B"},
				{validator: "A", voted: "B", auth: false},
			},
			results: []string{"A", "B"},
		}, {
			// Deauthorizing multiple accounts concurrently is permitted
			validators: []string{"A", "B", "C", "D"},
			votes: []testerVote{
				{validator: "A", voted: "C", auth: false},
				{validator: "B"},
				{validator: "C"},
				{validator: "A", voted: "D", auth: false},
				{validator: "B"},
				{validator: "C"},
				{validator: "A"},
				{validator: "B", voted: "D", auth: false},
				{validator: "C", voted: "D", auth: false},
				{validator: "A"},
				{validator: "B", voted: "C", auth: false},
			},
			results: []string{"A", "B"},
		}, {
			// Votes from deauthorized validators are discarded immediately (deauth votes)
			validators: []string{"A", "B", "C"},
			votes: []testerVote{
				{validator: "C", voted: "B", auth: false},
				{validator: "A", voted: "C", auth: false},
				{validator: "B", voted: "C", auth: false},
				{validator: "A", voted: "B", auth: false},
			},
			results: []string{"A", "B"},
		}, {
			// Votes from deauthorized validators are discarded immediately (auth votes)
			validators: []string{"A", "B", "C"},
			votes: []testerVote{
				{validator: "C", voted: "B", auth: false},
				{validator: "A", voted: "C", auth: false},
				{validator: "B", voted: "C", auth: false},
				{validator: "A", voted: "B", auth: false},
			},
			results: []string{"A", "B"},
		}, {
			// Cascading changes are not allowed, only the the account being voted on may change
			validators: []string{"A", "B", "C", "D"},
			votes: []testerVote{
				{validator: "A", voted: "C", auth: false},
				{validator: "B"},
				{validator: "C"},
				{validator: "A", voted: "D", auth: false},
				{validator: "B", voted: "C", auth: false},
				{validator: "C"},
				{validator: "A"},
				{validator: "B", voted: "D", auth: false},
				{validator: "C", voted: "D", auth: false},
			},
			results: []string{"A", "B", "C"},
		}, {
			// Changes reaching consensus out of bounds (via a deauth) execute on touch
			validators: []string{"A", "B", "C", "D"},
			votes: []testerVote{
				{validator: "A", voted: "C", auth: false},
				{validator: "B"},
				{validator: "C"},
				{validator: "A", voted: "D", auth: false},
				{validator: "B", voted: "C", auth: false},
				{validator: "C"},
				{validator: "A"},
				{validator: "B", voted: "D", auth: false},
				{validator: "C", voted: "D", auth: false},
				{validator: "A"},
				{validator: "C", voted: "C", auth: true},
			},
			results: []string{"A", "B"},
		}, {
			// Changes reaching consensus out of bounds (via a deauth) may go out of consensus on first touch
			validators: []string{"A", "B", "C", "D"},
			votes: []testerVote{
				{validator: "A", voted: "C", auth: false},
				{validator: "B"},
				{validator: "C"},
				{validator: "A", voted: "D", auth: false},
				{validator: "B", voted: "C", auth: false},
				{validator: "C"},
				{validator: "A"},
				{validator: "B", voted: "D", auth: false},
				{validator: "C", voted: "D", auth: false},
				{validator: "A"},
				{validator: "B", voted: "C", auth: true},
			},
			results: []string{"A", "B", "C"},
		}, {
			// Ensure that pending votes don't survive authorization status changes. This
			// corner case can only appear if a validator is quickly added, remove and then
			// readded (or the inverse), while one of the original voters dropped. If a
			// past vote is left cached in the system somewhere, this will interfere with
			// the final validator outcome.
			validators: []string{"A", "B", "C", "D", "E"},
			votes: []testerVote{
				{validator: "A", voted: "F", auth: true}, // Authorize F, 3 votes needed
				{validator: "B", voted: "F", auth: true},
				{validator: "C", voted: "F", auth: true},
				{validator: "D", voted: "F", auth: false}, // Deauthorize F, 4 votes needed (leave A's previous vote "unchanged")
				{validator: "E", voted: "F", auth: false},
				{validator: "B", voted: "F", auth: false},
				{validator: "C", voted: "F", auth: false},
				{validator: "D", voted: "F", auth: true}, // Almost authorize F, 2/3 votes needed
				{validator: "E", voted: "F", auth: true},
				{validator: "B", voted: "A", auth: false}, // Deauthorize A, 3 votes needed
				{validator: "C", voted: "A", auth: false},
				{validator: "D", voted: "A", auth: false},
				{validator: "B", voted: "F", auth: true}, // Finish authorizing F, 3/3 votes needed
			},
			results: []string{"B", "C", "D", "E", "F"},
		}, {
			// Epoch transitions reset all votes to allow chain checkpointing
			epoch:      3,
			validators: []string{"A", "B"},
			votes: []testerVote{
				{validator: "A", voted: "C", auth: true},
				{validator: "B"},
				{validator: "A"}, // Checkpoint block, (don't vote here, it's validated outside of snapshots)
				{validator: "B", voted: "C", auth: true},
			},
			results: []string{"A", "B"},
		},
	}
	// Run through the scenarios and test them
	for i, tt := range tests {
		// Create the account pool and generate the initial set of validators
		accounts := newTesterAccountPool()

		validators := make([]common.Address, len(tt.validators))
		for j, validator := range tt.validators {
			validators[j] = accounts.address(validator)
		}
		for j := 0; j < len(validators); j++ {
			for k := j + 1; k < len(validators); k++ {
				if bytes.Compare(validators[j][:], validators[k][:]) > 0 {
					validators[j], validators[k] = validators[k], validators[j]
				}
			}
		}
		// Create the genesis block with the initial set of validators
		db, depose := leveldb.NewTestDatabase()
		defer depose()
		bcStore := store.NewCachedStore(store.NewBlockchainDatabase(db))
		genesis := core2.GetGenesis(core2.NewGenesisInfo(nil, defaultDifficulty.Int64(), 0, big.NewInt(0), types.IstanbulConsensus, validators))
		if err := genesis.InitializeAndValidate(bcStore, db); err != nil {
			panic(err)
		}

		config := istanbul.DefaultConfig
		if tt.epoch != 0 {
			config.Epoch = tt.epoch
		}
		engine := New(config, accounts.accounts[tt.validators[0]], db).(*backend)
		chain, err := core2.NewBlockchain(bcStore, db, "", engine, nil)

		// Assemble a chain of headers from the cast votes
		headers := make([]*types.BlockHeader, len(tt.votes))
		for j, vote := range tt.votes {
			headers[j] = &types.BlockHeader{
				Height:          uint64(j) + 1,
				CreateTimestamp: big.NewInt(int64(j) * int64(config.BlockPeriod)),
				Creator:         accounts.address(vote.voted),
				Difficulty:      defaultDifficulty,
				Witness:         make([]byte, istanbul.WitnessSize),
				Consensus:       types.IstanbulConsensus,
			}
			extra, _ := prepareExtra(headers[j], validators)
			headers[j].ExtraData = extra
			if j > 0 {
				headers[j].PreviousBlockHash = headers[j-1].Hash()
			}
			if vote.auth {
				copy(headers[j].Witness[:], nonceAuthVote)
			}

			copy(headers[j].ExtraData, chain.Genesis().Header.ExtraData)
			accounts.sign(headers[j], vote.validator)
		}
		// Pass all the headers through clique and ensure tallying succeeds
		head := headers[len(headers)-1]

		snap, err := engine.snapshot(chain, head.Height, head.Hash(), headers)
		if err != nil {
			t.Errorf("test %d: failed to create voting snapshot: %v", i, err)
			continue
		}
		// Verify the final list of validators against the expected ones
		validators = make([]common.Address, len(tt.results))
		for j, validator := range tt.results {
			validators[j] = accounts.address(validator)
		}
		for j := 0; j < len(validators); j++ {
			for k := j + 1; k < len(validators); k++ {
				if bytes.Compare(validators[j][:], validators[k][:]) > 0 {
					validators[j], validators[k] = validators[k], validators[j]
				}
			}
		}
		result := snap.validators()
		if len(result) != len(validators) {
			t.Errorf("test %d: validators mismatch: have %x, want %x", i, result, validators)
			continue
		}
		for j := 0; j < len(result); j++ {
			if !bytes.Equal(result[j][:], validators[j][:]) {
				t.Errorf("test %d, validator %d: validator mismatch: have %x, want %x", i, j, result[j], validators[j])
			}
		}
	}
}

func TestSaveAndLoad(t *testing.T) {
	snap := &Snapshot{
		Epoch:  5,
		Height: 10,
		Hash:   common.EmptyHash,
		Votes: []*Vote{
			{
				Validator: common.BytesToAddress([]byte{1}),
				Block:     15,
				Address:   common.BytesToAddress([]byte{2}),
				Authorize: false,
			},
		},
		Tally: map[common.Address]Tally{
			common.BytesToAddress([]byte{3}): Tally{
				Authorize: false,
				Votes:     20,
			},
		},
		ValSet: validator.NewSet([]common.Address{
			common.BytesToAddress([]byte{4}),
			common.BytesToAddress([]byte{5}),
		}, istanbul.RoundRobin),
	}
	db, _ := leveldb.NewTestDatabase()
	err := snap.store(db)
	if err != nil {
		t.Errorf("store snapshot failed: %v", err)
	}

	snap1, err := loadSnapshot(snap.Epoch, db, snap.Hash)
	if err != nil {
		t.Errorf("load snapshot failed: %v", err)
	}
	if snap.Epoch != snap1.Epoch {
		t.Errorf("epoch mismatch: have %v, want %v", snap1.Epoch, snap.Epoch)
	}
	if snap.Hash != snap1.Hash {
		t.Errorf("hash mismatch: have %v, want %v", snap1.Height, snap.Height)
	}
	if !reflect.DeepEqual(snap.Votes, snap.Votes) {
		t.Errorf("votes mismatch: have %v, want %v", snap1.Votes, snap.Votes)
	}
	if !reflect.DeepEqual(snap.Tally, snap.Tally) {
		t.Errorf("tally mismatch: have %v, want %v", snap1.Tally, snap.Tally)
	}
	if !reflect.DeepEqual(snap.ValSet, snap.ValSet) {
		t.Errorf("validator set mismatch: have %v, want %v", snap1.ValSet, snap.ValSet)
	}
}
