/**
*  @file
*  @copyright defined in scdo/LICENSE
 */

package downloader

import (
	"errors"
	"math/big"
	"sync"
	"time"

	"github.com/elcn233/go-scdo/common"
	"github.com/elcn233/go-scdo/log"
	"github.com/elcn233/go-scdo/p2p"
)

// MsgWaitTimeout this timeout should not be happened, but we need to handle it in case of such errors.
const MsgWaitTimeout = time.Second * 25
const maxLoopAllowed = 100

var (
	errReceivedQuitMsg = errors.New("Received quit msg")
	errPeerQuit        = errors.New("Peer quit")
)

// Peer define some interfaces that request peer data
type Peer interface {
	Head() (common.Hash, *big.Int)
	RequestHeadersByHashOrNumber(magic uint32, origin common.Hash, num uint64, amount int, reverse bool) error
	RequestBlocksByHashOrNumber(magic uint32, origin common.Hash, num uint64, amount int) error
	GetPeerRequestInfo() (uint32, common.Hash, uint64, int)
	DisconnectPeer(reason string)
}

type peerConn struct {
	peerID         string
	peer           Peer
	waitingMsgMap  map[uint16]chan *p2p.Message
	lockForWaiting sync.RWMutex

	log    *log.ScdoLog
	quitCh chan struct{}
}

func newPeerConn(p Peer, peerID string, log *log.ScdoLog) *peerConn {
	return &peerConn{
		peerID:        peerID,
		peer:          p,
		waitingMsgMap: make(map[uint16]chan *p2p.Message),
		log:           log,
		quitCh:        make(chan struct{}),
	}
}

func (p *peerConn) close() {
	close(p.quitCh)
}

func (p *peerConn) waitMsg(magic uint32, msgCode uint16, cancelCh chan struct{}) (ret interface{}, err error) {
	rcvCh := make(chan *p2p.Message)
	p.lockForWaiting.Lock()
	p.waitingMsgMap[msgCode] = rcvCh
	p.lockForWaiting.Unlock()
	timeout := time.NewTimer(MsgWaitTimeout)
	//timeout := time.Timer(MsgWaitTimeout)
	loopCount := 0
Again:

	select {

	case <-p.quitCh:
		err = errPeerQuit
	case <-cancelCh:
		err = errReceivedQuitMsg
	case msg := <-rcvCh:
		switch msgCode {
		case BlockHeadersMsg:
			var reqMsg BlockHeadersMsgBody
			if err := common.Deserialize(msg.Payload, &reqMsg); err != nil {
				loopCount++
				if loopCount > maxLoopAllowed {
					break Again
				}
				goto Again
			}
			if reqMsg.Magic != magic {
				p.log.Debug("Downloader.waitMsg  BlockHeadersMsg MAGIC_NOT_MATCH msg=%s, magic=%d, pid=%s", CodeToStr(msgCode), magic, p.peerID)
				loopCount++
				if loopCount > maxLoopAllowed {
					break Again
				}
				goto Again
			}
			ret = reqMsg.Headers
		case BlocksMsg:
			var reqMsg BlocksMsgBody
			if err := common.Deserialize(msg.Payload, &reqMsg); err != nil {
				loopCount++
				if loopCount > maxLoopAllowed {
					break Again
				}
				goto Again
			}
			if reqMsg.Magic != magic {
				p.log.Debug("Downloader.waitMsg  BlocksMsg MAGIC_NOT_MATCH msg=%s pid=%s", CodeToStr(msgCode), p.peerID)
				loopCount++
				if loopCount > maxLoopAllowed {
					break Again
				}
				goto Again
			}

			ret = reqMsg.Blocks
		}
	case <-timeout.C:
		p.log.Debug("Downloader.waitMsg  timeout msg=%s pid=%s", CodeToStr(msgCode), p.peerID)
		//err = fmt.Errorf("Download.peerconn wait for msg %s timeout.magic= %d ip= %s", CodeToStr(msgCode), magic, p.peerID)
		err = errReceivedQuitMsg
	}

	p.lockForWaiting.Lock()
	delete(p.waitingMsgMap, msgCode)
	p.lockForWaiting.Unlock()
	close(rcvCh)
	return ret, err
}

func (p *peerConn) deliverMsg(msgCode uint16, msg *p2p.Message) {
	defer func() {
		if recover() != nil {
			p.log.Info("peerConn.deliverMsg PANIC msg=%s pid=%s", CodeToStr(msgCode), p.peerID)
		}
	}()

	p.lockForWaiting.Lock()
	ch, ok := p.waitingMsgMap[msgCode]
	p.lockForWaiting.Unlock()
	if !ok {
		return
	}
	ch <- msg
}
