/**
*  @file
*  @copyright defined in scdo/LICENSE
 */

package discovery

import (
	"net"
	"testing"

	"github.com/elcn233/go-scdo/crypto"
	"github.com/stretchr/testify/assert"
)

func Test_Server_StartService(t *testing.T) {
	nodeDir := "."
	myID := *crypto.MustGenerateShardAddress(1)
	myAddr, _ := net.ResolveUDPAddr("udp", "127.0.0.1:9777")
	bootstrap := make([]*Node, 0)
	shard := uint(1)

	db, _ := StartService(nodeDir, myID, myAddr, bootstrap, shard)
	assert.Equal(t, db != nil, true)
}
