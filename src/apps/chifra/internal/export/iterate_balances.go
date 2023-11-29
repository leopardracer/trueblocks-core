package exportPkg

import (
	"context"
	"fmt"
	"math/big"
	"sort"

	"github.com/TrueBlocks/trueblocks-core/src/apps/chifra/pkg/base"
	"github.com/TrueBlocks/trueblocks-core/src/apps/chifra/pkg/filter"
	"github.com/TrueBlocks/trueblocks-core/src/apps/chifra/pkg/logger"
	"github.com/TrueBlocks/trueblocks-core/src/apps/chifra/pkg/monitor"
	"github.com/TrueBlocks/trueblocks-core/src/apps/chifra/pkg/types"
	"github.com/TrueBlocks/trueblocks-core/src/apps/chifra/pkg/utils"
)

func (opts *ExportOptions) readBalances(
	mon *monitor.Monitor,
	filter *filter.AppearanceFilter,
	errorChan chan error,
) ([]*types.SimpleToken, error) {
	testMode := opts.Globals.TestMode
	nErrors := 0
	var cnt int
	var err error
	var appMap map[types.SimpleAppearance]*types.SimpleToken
	if appMap, cnt, err = monitor.AsMap[types.SimpleToken](mon, filter); err != nil {
		errorChan <- err
		return nil, err
	} else if opts.NoZero && cnt == 0 {
		errorChan <- fmt.Errorf("no appearances found for %s", mon.Address.Hex())
		return nil, nil
	}

	bar := logger.NewBar(logger.BarOptions{
		Prefix:  mon.Address.Hex(),
		Enabled: !opts.Globals.TestMode,
		Total:   mon.Count(),
	})

	iterFunc := func(app types.SimpleAppearance, value *types.SimpleToken) error {
		var balance *big.Int
		if balance, err = opts.Conn.GetBalanceByAppearance(mon.Address, &app); err != nil {
			return err
		}

		value.Address = base.FAKE_ETH_ADDRESS
		value.Holder = mon.Address
		value.BlockNumber = uint64(app.BlockNumber)
		value.TransactionIndex = uint64(app.TransactionIndex)
		value.Balance = *balance
		value.Timestamp = app.Timestamp
		bar.Tick()

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

	// Sort the items back into an ordered array by block number
	items := make([]*types.SimpleToken, 0, len(appMap))
	for _, tx := range appMap {
		items = append(items, tx)
	}
	sort.Slice(items, func(i, j int) bool {
		if opts.Reversed {
			i, j = j, i
		}
		return items[i].BlockNumber < items[j].BlockNumber
	})

	// Return the array of items
	return items, nil
}
