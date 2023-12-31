/**
*  @file
*  @copyright defined in scdo/LICENSE
 */

package backend

import (
	"testing"

	"github.com/elcn233/go-scdo/common"
	"github.com/elcn233/go-scdo/crypto"
	"github.com/elcn233/go-scdo/p2p"
	lru "github.com/hashicorp/golang-lru"
)

func TestIstanbulMessage(t *testing.T) {
	_, backend := newBlockChain(1)

	// generate one msg
	data := []byte("data1")
	hash := crypto.HashBytes(data)
	msg := makeMsg(istanbulMsg, data)
	addr := *crypto.MustGenerateRandomAddress()

	// 1. this message should not be in cache
	// for peers
	if _, ok := backend.recentMessages.Get(addr); ok {
		t.Fatalf("the cache of messages for this peer should be nil")
	}

	// for self
	if _, ok := backend.knownMessages.Get(hash); ok {
		t.Fatalf("the cache of messages should be nil")
	}

	// 2. this message should be in cache after we handle it
	_, err := backend.HandleMsg(addr, msg)
	if err != nil {
		t.Fatalf("handle message failed: %v", err)
	}
	// for peers
	if ms, ok := backend.recentMessages.Get(addr); ms == nil || !ok {
		t.Fatalf("the cache of messages for this peer cannot be nil")
	} else if m, ok := ms.(*lru.ARCCache); !ok {
		t.Fatalf("the cache of messages for this peer cannot be casted")
	} else if _, ok := m.Get(hash); !ok {
		t.Fatalf("the cache of messages for this peer cannot be found")
	}

	// for self
	if _, ok := backend.knownMessages.Get(hash); !ok {
		t.Fatalf("the cache of messages cannot be found")
	}
}

func makeMsg(msgcode uint16, data interface{}) p2p.Message {
	payload := common.SerializePanic(data)
	return p2p.Message{Code: msgcode, Payload: payload}
}
