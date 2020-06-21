package loadgen

import (
	"testing"

	"github.com/Factom-Asset-Tokens/factom"
	"github.com/PaulBernier/chockagent/common"
	"github.com/stretchr/testify/require"
)

const ENTRY_HEADER_LENGTH = 35

func TestExactEntryLength(t *testing.T) {
	require := require.New(t)

	esAddress, _ := factom.NewEsAddress("Es3ytEKt6R5jM9juC4ks7EgxQSX8BpRnM4WADtgFoq7j1WgbEEGW")

	size := 1024
	composer, err := NewRandomEntryComposer(
		[]string{"2d98021e3cf71580102224b2fcb4c5c60595e8fdf6fd1b97c6ef63e9fb3ed635"}, esAddress, common.IntRange{Min: size, Max: size})

	_, reveal, err := composer.Compose()

	require.NoError(err)
	require.Len(reveal, ENTRY_HEADER_LENGTH+size)
}

func TestRangeEntryLength(t *testing.T) {
	require := require.New(t)

	esAddress, _ := factom.NewEsAddress("Es3ytEKt6R5jM9juC4ks7EgxQSX8BpRnM4WADtgFoq7j1WgbEEGW")

	min := 1024
	max := 2048
	composer, err := NewRandomEntryComposer(
		[]string{"2d98021e3cf71580102224b2fcb4c5c60595e8fdf6fd1b97c6ef63e9fb3ed635"}, esAddress, common.IntRange{Min: min, Max: max})

	_, reveal, err := composer.Compose()

	require.NoError(err)
	require.GreaterOrEqual(len(reveal), ENTRY_HEADER_LENGTH+min)
	require.LessOrEqual(len(reveal), ENTRY_HEADER_LENGTH+max)
}
