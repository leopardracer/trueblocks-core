// Copyright 2021 The TrueBlocks Authors. All rights reserved.
// Use of this source code is governed by a license that can
// be found in the LICENSE file.

package blocksPkg

import (
	"context"
	"errors"
	"sort"

	"github.com/TrueBlocks/trueblocks-core/src/apps/chifra/pkg/identifiers"
	"github.com/TrueBlocks/trueblocks-core/src/apps/chifra/pkg/logger"
	"github.com/TrueBlocks/trueblocks-core/src/apps/chifra/pkg/output"
	"github.com/TrueBlocks/trueblocks-core/src/apps/chifra/pkg/types"
	"github.com/TrueBlocks/trueblocks-core/src/apps/chifra/pkg/utils"
	"github.com/ethereum/go-ethereum"
)

func (opts *BlocksOptions) HandleHashes() error {
	chain := opts.Globals.Chain
	testMode := opts.Globals.TestMode
	nErrors := 0

	ctx, cancel := context.WithCancel(context.Background())
	fetchData := func(modelChan chan types.Modeler[types.RawBlock], errorChan chan error) {
		// var cnt int
		var err error
		var appMap map[types.SimpleAppearance]*types.SimpleBlock[string]
		if appMap, _, err = identifiers.AsMap[types.SimpleBlock[string]](chain, opts.BlockIds); err != nil {
			errorChan <- err
			cancel()
		}

		bar := logger.NewBar(logger.BarOptions{
			Type:    logger.Expanding,
			Enabled: !opts.Globals.TestMode,
			Total:   int64(len(appMap)),
		})

		iterFunc := func(app types.SimpleAppearance, value *types.SimpleBlock[string]) error {
			bn := uint64(app.BlockNumber)
			if block, err := opts.Conn.GetBlockHeaderByNumber(bn); err != nil {
				errorChan <- err
				if errors.Is(err, ethereum.NotFound) {
					errorChan <- errors.New("uncles not found")
				}
				cancel()
				return nil
			} else {
				bar.Tick()
				*value = block
			}
			return nil
		}

		iterErrorChan := make(chan error)
		iterCtx, iterCancel := context.WithCancel(context.Background())
		defer iterCancel()
		go utils.IterateOverMap(iterCtx, iterErrorChan, appMap, iterFunc)
		for err := range iterErrorChan {
			if !testMode || nErrors == 0 {
				errorChan <- err
				nErrors++
			}
		}
		bar.Finish(true)

		items := make([]*types.SimpleBlock[string], 0, len(appMap))
		for _, item := range appMap {
			items = append(items, item)
		}
		sort.Slice(items, func(i, j int) bool {
			if items[i].BlockNumber == items[j].BlockNumber {
				return items[i].Hash.Hex() < items[j].Hash.Hex()
			}
			return items[i].BlockNumber < items[j].BlockNumber
		})

		for _, item := range items {
			item := item
			modelChan <- item
		}
	}

	extra := map[string]interface{}{
		"hashes":     opts.Hashes,
		"uncles":     opts.Uncles,
		"articulate": opts.Articulate,
	}

	return output.StreamMany(ctx, fetchData, opts.Globals.OutputOptsWithExtra(extra))
}
