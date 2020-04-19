package loadgen

import (
	"crypto/ed25519"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"math/rand"
	"time"

	"github.com/PaulBernier/chockagent/common"

	"github.com/Factom-Asset-Tokens/factom"
)

const (
	EntryCommitSize = 1 + // version
		6 + // timestamp
		32 + // entry hash
		1 + // ec cost
		32 + // ec pub
		64 // sig,
	EntryHeaderSize = 1 + // version
		32 + // chain id
		2 // total len
)

type RandomEntryComposer struct {
	chainIDs           [][]byte
	privateKey         ed25519.PrivateKey
	publicKey          ed25519.PublicKey
	entrySizeGenerator func() int
}

func NewRandomEntryComposer(chainIDsStr []string,
	esAddress factom.EsAddress,
	entrySizeRange common.IntRange) (*RandomEntryComposer, error) {
	comp := new(RandomEntryComposer)

	chainIDs := make([][]byte, len(chainIDsStr))

	for i, chainIDStr := range chainIDsStr {
		chainID, err := hex.DecodeString(chainIDStr)
		if err != nil {
			return nil, err
		}
		chainIDs[i] = chainID
	}

	comp.chainIDs = chainIDs
	comp.privateKey = esAddress.PrivateKey()
	comp.publicKey = esAddress.PublicKey()

	if entrySizeRange.Min < 32 || entrySizeRange.Min > entrySizeRange.Max {
		return nil, fmt.Errorf("Invalid entry size range: [%+v]", entrySizeRange)
	}

	if entrySizeRange.Min == entrySizeRange.Max {
		comp.entrySizeGenerator = func() int { return entrySizeRange.Min }
	} else {
		comp.entrySizeGenerator = func() int {
			return entrySizeRange.Min + rand.Intn(entrySizeRange.Max-entrySizeRange.Min)
		}
	}

	return comp, nil
}

func (comp *RandomEntryComposer) Compose() ([]byte, []byte, error) {
	content := make([]byte, comp.entrySizeGenerator())
	_, err := rand.Read(content)
	if err != nil {
		return nil, nil, err
	}

	chainID := comp.chainIDs[rand.Intn(len(comp.chainIDs))]
	reveal := entryBytes(chainID, content)
	commit := generateCommit(reveal, comp.publicKey, comp.privateKey)

	return commit, reveal, nil
}

func entryBytes(chainID, content []byte) []byte {
	data := make([]byte, len(content)+EntryHeaderSize)
	i := 1
	i += copy(data[i:], chainID[:])
	binary.BigEndian.PutUint16(data[i:i+2], uint16(0))
	i += 2

	copy(data[i:], content)

	return data
}

func generateCommit(entrydata []byte, publicKey ed25519.PublicKey, privateKey ed25519.PrivateKey) []byte {
	commit := make([]byte, EntryCommitSize)

	i := 1 // Skip version byte

	ms := time.Now().Unix() * 1e3
	putInt48BE(commit[i:], ms)
	i += 6

	// Entry Hash
	hash := computeEntryHash(entrydata)
	i += copy(commit[i:], hash[:])

	cost, err := entryCost(len(entrydata))

	if err != nil {
		panic(fmt.Sprintf("Failed to compute entry cost for entry of length [%d]: [%s]",
			len(entrydata), err))
	}

	commit[i] = byte(cost)
	i++

	// Public Key
	signedDataSize := i
	i += copy(commit[i:], publicKey)

	// Signature
	sig := ed25519.Sign(privateKey, commit[:signedDataSize])
	copy(commit[i:], sig)

	return commit
}

func computeEntryHash(data []byte) [32]byte {
	sum := sha512.Sum512(data)
	saltedSum := make([]byte, len(sum)+len(data))
	i := copy(saltedSum, sum[:])
	copy(saltedSum[i:], data)
	return sha256.Sum256(saltedSum)
}

func putInt48BE(data []byte, x int64) {
	const size = 6
	for i := 0; i < size; i++ {
		data[i] = byte(x >> (8 * (size - 1 - i)))
	}
}

func entryCost(size int) (uint8, error) {
	if size < EntryHeaderSize {
		return 0, fmt.Errorf("invalid size")
	}
	size -= EntryHeaderSize
	if size > 10240 {
		return 0, fmt.Errorf("Entry cannot be larger than 10KB")
	}
	cost := uint8(size / 1024)
	if size%1024 > 0 {
		cost++
	}
	if cost < 1 {
		cost = 1
	}

	return cost, nil
}
