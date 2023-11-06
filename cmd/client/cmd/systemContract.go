/**
*  @file
*  @copyright defined in scdo/LICENSE
 */

package cmd

import (
	"errors"
	"fmt"

	"github.com/elcn233/go-scdo/cmd/util"
	"github.com/elcn233/go-scdo/common"
	"github.com/elcn233/go-scdo/common/hexutil"
	"github.com/elcn233/go-scdo/core/types"
	"github.com/elcn233/go-scdo/rpc"
)

type handler func(client *rpc.Client) (interface{}, interface{}, error)

var (
	errInvalidCommand    = errors.New("invalid command")
	errInvalidSubcommand = errors.New("invalid subcommand")

	systemContract = map[string]map[string]handler{
		"htlc": map[string]handler{
			"create":   createHTLC,
			"withdraw": withdraw,
			"refund":   refund,
			"get":      getHTLC,
		},
		"domain": map[string]handler{
			"create":   createDomainName,
			"getOwner": getDomainNameOwner,
		},
		"subchain": map[string]handler{
			"register": registerSubChain,
			"query":    querySubChain,
		},
	}

	// if the method have key-value, use the call method to get receipt
	callFlags = map[string]map[string]string{
		"htlc": map[string]string{
			"get": "1",
		},
	}
)

// sendSystemContractTx send system contract transaction
func sendSystemContractTx(client *rpc.Client, to common.Address, method byte, payload []byte) (*types.Transaction, error) {
	key, txd, err := makeTransactionData(client)
	if err != nil {
		return nil, err
	}

	txd.To = to
	txd.Payload = append([]byte{method}, payload...)

	tx, err := util.GenerateTx(key.PrivateKey, &txd.From, txd.To, txd.Amount, txd.GasPrice, txd.GasLimit, txd.AccountNonce, txd.Payload)
	if err != nil {
		return nil, err
	}

	return tx, err
}

// sendTx send transaction or contract
func sendTx(client *rpc.Client, arg interface{}) error {
	var result bool
	if err := client.Call(&result, "scdo_addTx", arg); err != nil || !result {
		return fmt.Errorf("Failed to call rpc, %s", err)
	}

	return nil
}

// callTx call transaction or contract
func callTx(client *rpc.Client, tx *types.Transaction) (interface{}, error) {
	var result interface{}
	if tx != nil {
		if err := client.Call(&result, "scdo_call", tx.Data.To.Hex(), hexutil.BytesToHex(tx.Data.Payload), -1); err != nil {
			return nil, fmt.Errorf("Failed to call rpc, %s", err)
		}
	} else {
		return nil, errors.New("Invalid parameters")
	}

	return result, nil
}
