package vrfkey

import (
	"crypto/ecdsa"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"math/big"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/chainlink/v2/core/services/signatures/secp256k1"
)

func TestVRFKeys_KeyV2_Raw(t *testing.T) {
	privK, err := ecdsa.GenerateKey(crypto.S256(), rand.Reader)
	require.NoError(t, err)

	r := Raw(privK.D.Bytes())

	assert.Equal(t, r.String(), r.GoString())
	assert.Equal(t, "<VRF Raw Private Key>", r.String())
}

func TestVRFKeys_KeyV2(t *testing.T) {
	k, err := NewV2()
	require.NoError(t, err)

	assert.Equal(t, hexutil.Encode(k.PublicKey[:]), k.ID())
	assert.Equal(t, Raw(secp256k1.ToInt(*k.k).Bytes()), k.Raw())

	t.Run("generates proof", func(t *testing.T) {
		p, err := k.GenerateProof(big.NewInt(1))

		assert.NotZero(t, p)
		assert.NoError(t, err)
	})
}

// keyHash & publicKey for vrf job
func TestVRFKeys_KeyV2_XY(t *testing.T) {
	uncompressedPubKey := "0x88ec0d28f83ad969e249604d3d7eeabb478603996e3e26d1260c8534c93719a255b10afad26ea87daed9081c09743925728f56783219013eed3420b410506268"
	// Put key in ECDSA format
	if strings.HasPrefix(uncompressedPubKey, "0x") {
		uncompressedPubKey = strings.Replace(uncompressedPubKey, "0x", "04", 1)
	}
	pubBytes, err := hex.DecodeString(uncompressedPubKey)
	pkTmp, err := crypto.UnmarshalPubkey(pubBytes)
	fmt.Printf("pkTmp: %v, err: %v\n", pkTmp, err)
	if pkTmp != nil {
		fmt.Printf("x: %v, y: %v\n", pkTmp.X, pkTmp.Y)
	}
}
