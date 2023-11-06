/**
*  @file
*  @copyright defined in scdo/LICENSE
 */

package core

import (
	"github.com/elcn233/go-scdo/common"
)

func (c *core) handleFinalCommitted() error {
	c.logger.Debug("Received a final committed proposal")
	c.startNewRound(common.Big0)
	return nil
}
