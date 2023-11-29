package statePkg

import (
	"context"
	"fmt"
	"sort"

	"github.com/TrueBlocks/trueblocks-core/src/apps/chifra/pkg/articulate"
	"github.com/TrueBlocks/trueblocks-core/src/apps/chifra/pkg/base"
	"github.com/TrueBlocks/trueblocks-core/src/apps/chifra/pkg/call"
	"github.com/TrueBlocks/trueblocks-core/src/apps/chifra/pkg/identifiers"
	"github.com/TrueBlocks/trueblocks-core/src/apps/chifra/pkg/logger"
	"github.com/TrueBlocks/trueblocks-core/src/apps/chifra/pkg/output"
	"github.com/TrueBlocks/trueblocks-core/src/apps/chifra/pkg/types"
	"github.com/TrueBlocks/trueblocks-core/src/apps/chifra/pkg/utils"
)

func (opts *StateOptions) HandleCall() error {
	chain := opts.Globals.Chain
	testMode := opts.Globals.TestMode
	nErrors := 0

	artFunc := func(str string, function *types.SimpleFunction) error {
		return articulate.ArticulateFunction(function, "", str[2:])
	}

	callAddress := base.HexToAddress(opts.Addrs[0])
	if opts.ProxyFor != "" {
		callAddress = base.HexToAddress(opts.ProxyFor)
	}

	ctx, cancel := context.WithCancel(context.Background())
	fetchData := func(modelChan chan types.Modeler[types.RawResult], errorChan chan error) {
		// var cnt int
		var err error
		var appMap map[types.SimpleAppearance]*types.SimpleResult
		if appMap, _, err = identifiers.AsMap[types.SimpleResult](chain, opts.BlockIds); err != nil {
			errorChan <- err
			cancel()
		} else {
			bar := logger.NewBar(logger.BarOptions{
				Enabled: !opts.Globals.TestMode,
				Total:   int64(len(appMap)),
			})

			iterFunc := func(app types.SimpleAppearance, value *types.SimpleResult) error {
				bn := uint64(app.BlockNumber)
				if contractCall, _, err := call.NewContractCall(opts.Conn, callAddress, opts.Call); err != nil {
					wrapped := fmt.Errorf("the --call value provided (%s) was not found: %s", opts.Call, err)
					errorChan <- wrapped
					cancel()
				} else {
					contractCall.BlockNumber = bn
					results, err := contractCall.Call(artFunc)
					if err != nil {
						errorChan <- err
						cancel()
					} else {
						bar.Tick()
						*value = *results
					}
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

			items := make([]types.SimpleResult, 0, len(appMap))
			for _, v := range appMap {
				v := v
				items = append(items, *v)
			}
			sort.Slice(items, func(i, j int) bool {
				return items[i].BlockNumber < items[j].BlockNumber
			})

			for _, item := range items {
				item := item
				modelChan <- &item
			}
		}
	}

	extra := map[string]interface{}{
		"articulate": opts.Articulate,
	}

	return output.StreamMany(ctx, fetchData, opts.Globals.OutputOptsWithExtra(extra))
}
