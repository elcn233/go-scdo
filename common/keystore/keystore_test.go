/**
*  @file
*  @copyright defined in scdo/LICENSE
 */

package keystore

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/elcn233/go-scdo/crypto"
	"github.com/stretchr/testify/assert"
)

func Test_KeyStore(t *testing.T) {
	dir, err := ioutil.TempDir("", "keystore")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(dir)

	addr, keypair, err := crypto.GenerateKeyPair()
	if err != nil {
		panic(err)
	}

	key := &Key{
		*addr,
		keypair,
	}

	password := "testfile"
	fileName := filepath.Join(dir, "keyfile")
	err = StoreKey(fileName, password, key)
	assert.Equal(t, err, nil)

	result, err := GetKey(fileName, password)
	assert.Equal(t, err, nil)
	assert.Equal(t, crypto.FromECDSA(key.PrivateKey), crypto.FromECDSA(result.PrivateKey))
	assert.Equal(t, key.Address, result.Address)
}
