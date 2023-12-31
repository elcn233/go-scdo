/**
*  @file
*  @copyright defined in scdo/LICENSE
 */

package istanbul

import (
	"crypto/ecdsa"

	"github.com/elcn233/go-scdo/common"
	"github.com/elcn233/go-scdo/crypto"
)

const WitnessSize = 8

// GetSignatureAddress gets the signer address from the signature
func GetSignatureAddress(data []byte, sig []byte) (common.Address, error) {
	// 1. Keccak data
	hashData := crypto.Keccak256([]byte(data))
	// 2. Recover public key
	var pubkey *ecdsa.PublicKey
	pubkey, err := crypto.SigToPub(hashData, sig)
	if err != nil {
		return common.Address{}, err
	}
	return *crypto.PubkeyToAddress(*pubkey), nil
}

func CheckValidatorSignature(valSet ValidatorSet, data []byte, sig []byte) (common.Address, error) {
	// 1. Get signature address
	signer, err := GetSignatureAddress(data, sig)
	if err != nil {
		return common.Address{}, err
	}

	// 2. Check validator
	if _, val := valSet.GetByAddress(signer); val != nil {
		return val.Address(), nil
	}

	return common.Address{}, ErrUnauthorizedAddress
}
