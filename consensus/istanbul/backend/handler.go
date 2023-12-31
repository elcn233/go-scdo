/**
*  @file
*  @copyright defined in scdo/LICENSE
 */

package backend

import (
	"errors"

	"github.com/elcn233/go-scdo/common"
	"github.com/elcn233/go-scdo/consensus"
	"github.com/elcn233/go-scdo/consensus/istanbul"
	"github.com/elcn233/go-scdo/crypto"
	"github.com/elcn233/go-scdo/p2p"
	lru "github.com/hashicorp/golang-lru"
)

const (
	istanbulMsg = 0x11
)

var (
	// errDecodeFailed is returned when decode message fails
	errDecodeFailed = errors.New("fail to decode istanbul message")
)

// Protocol implements consensus.Engine.Protocol
func (sb *backend) Protocol() consensus.Protocol {
	return consensus.Protocol{
		Name:     "istanbul",
		Versions: []uint{64},
		Lengths:  []uint64{18},
	}
}

// HandleMsg implements consensus.Handler.HandleMsg
func (sb *backend) HandleMsg(addr common.Address, message interface{}) (bool, error) {
	sb.coreMu.Lock()
	defer sb.coreMu.Unlock()

	msg, ok := message.(p2p.Message)
	if !ok {
		return false, errDecodeFailed
	}

	if msg.Code == istanbulMsg {
		if !sb.coreStarted {
			return true, istanbul.ErrStoppedEngine
		}

		var data []byte
		if err := common.Deserialize(msg.Payload, &data); err != nil {
			return true, errDecodeFailed
		}

		hash := crypto.HashBytes(data)

		// Mark peer's message
		ms, ok := sb.recentMessages.Get(addr)
		var m *lru.ARCCache
		if ok {
			m, _ = ms.(*lru.ARCCache)
		} else {
			m, _ = lru.NewARC(inmemoryMessages)
			sb.recentMessages.Add(addr, m)
		}
		m.Add(hash, true)

		// Mark self known message
		if _, ok := sb.knownMessages.Get(hash); ok {
			return true, nil
		}
		sb.knownMessages.Add(hash, true)

		go sb.istanbulEventMux.Post(istanbul.MessageEvent{
			Payload: data,
		})

		return true, nil
	}
	return false, nil
}

// SetBroadcaster implements consensus.Handler.SetBroadcaster
func (sb *backend) SetBroadcaster(broadcaster consensus.Broadcaster) {
	sb.broadcaster = broadcaster
}

func (sb *backend) NewChainHead() error {
	sb.coreMu.RLock()
	defer sb.coreMu.RUnlock()
	if !sb.coreStarted {
		return istanbul.ErrStoppedEngine
	}
	go sb.istanbulEventMux.Post(istanbul.FinalCommittedEvent{})
	return nil
}
