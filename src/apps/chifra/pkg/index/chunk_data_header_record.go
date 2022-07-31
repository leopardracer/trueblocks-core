package index

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/TrueBlocks/trueblocks-core/src/apps/chifra/pkg/cache"
	"github.com/TrueBlocks/trueblocks-core/src/apps/chifra/pkg/file"
	"github.com/TrueBlocks/trueblocks-core/src/apps/chifra/pkg/unchained"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

// HeaderRecord is the first 44 bytes of an ChunkData. This structure carries a magic number (4 bytes),
// a version specifier (32 bytes), and two four-byte integers representing the number of records in each
// of the two tables.
type HeaderRecord struct {
	Magic           uint32
	Hash            common.Hash
	AddressCount    uint32
	AppearanceCount uint32
}

func (h *HeaderRecord) String() string {
	b, _ := json.Marshal(h)
	return string(b)
}

func readHeader(fl *os.File) (header HeaderRecord, err error) {
	err = binary.Read(fl, binary.LittleEndian, &header)
	if err != nil {
		return
	}

	// Because we call this frequently, we only check that the magic number is correct
	// we let the caller check the hash if needed
	if header.Magic != file.MagicNumber {
		return header, fmt.Errorf("magic number in file %s is incorrect, expected %d, got %d", fl.Name(), file.MagicNumber, header.Magic)
	}

	return
}

func ReadChunkHeader(chain, fileName string) (header HeaderRecord, err error) {
	fileName = ToIndexPath(fileName)
	ff, err := os.Open(fileName)
	if err != nil {
		return HeaderRecord{}, err
	}
	defer ff.Close()

	if header, err = readHeader(ff); err != nil {
		return
	}

	// TODO: BOGUS - DOES CHECKING FOR OLD INDEXES WORK?
	headerHash := hexutil.Encode(header.Hash.Bytes())
	hasZeroHash := headerHash == unchained.ZeroMagicHash
	hasMagicHash := headerHash == unchained.HeaderMagicHash

	// Since moving to the new unchained index code, we look for magicHash in the index file
	// with the following caveat. Index files on mainnet prior to block 13,000,000 have 0x000...000
	// in their headers. This will change when we go to version 1.0.0 of the index spec later.
	rng, err := cache.RangeFromFilename(fileName)
	if err != nil {
		return header, err
	}

	// We did not correct index chunks prior to 13,000,000 on mainnet
	if chain == "mainnet" && rng.First <= 13000000 {
		if !hasZeroHash {
			_, fileName = filepath.Split(fileName)
			return header, fmt.Errorf("expected headerHash of %s got %s in file %s", unchained.ZeroMagicHash, headerHash, fileName)
		}
	} else {
		if !hasMagicHash {
			return header, fmt.Errorf("header has incorrect hash in %s, expected %s, got %s", fileName, unchained.HeaderMagicHash, headerHash)
		}
	}

	return
}
