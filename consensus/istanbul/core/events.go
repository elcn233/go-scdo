/**
*  @file
*  @copyright defined in scdo/LICENSE
 */

package core

import (
	"github.com/elcn233/go-scdo/consensus/istanbul"
)

type backlogEvent struct {
	src istanbul.Validator
	msg *message
}

type timeoutEvent struct{}
