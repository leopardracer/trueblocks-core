package slurpPkg

/*-------------------------------------------------------------------------------------------
 * qblocks - fast, easily-accessible, fully-decentralized data from blockchains
 * copyright (c) 2016, 2021 TrueBlocks, LLC (http://trueblocks.io)
 *
 * This program is free software: you may redistribute it and/or modify it under the terms
 * of the GNU General Public License as published by the Free Software Foundation, either
 * version 3 of the License, or (at your option) any later version. This program is
 * distributed in the hope that it will be useful, but WITHOUT ANY WARRANTY; without even
 * the implied warranty of MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the GNU
 * General Public License for more details. You should have received a copy of the GNU General
 * Public License along with this program. If not, see http://www.gnu.org/licenses/.
 *-------------------------------------------------------------------------------------------*/
/*
 * This file was auto generated with makeClass --gocmds. DO NOT EDIT.
 */

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/TrueBlocks/trueblocks-core/src/apps/chifra/cmd/globals"
	"github.com/TrueBlocks/trueblocks-core/src/apps/chifra/pkg/logger"
)

type SlurpOptions struct {
	Addrs       []string
	Blocks      []string
	Types       []string
	Appearances bool
	Globals     globals.GlobalOptionsType
}

func (opts *SlurpOptions) TestLog() {
	logger.TestLog(len(opts.Addrs) > 0, "Addrs: ", opts.Addrs)
	logger.TestLog(len(opts.Blocks) > 0, "Blocks: ", opts.Blocks)
	logger.TestLog(len(opts.Types) > 0, "Types: ", opts.Types)
	logger.TestLog(opts.Appearances, "Appearances: ", opts.Appearances)
	opts.Globals.TestLog()
}

func (opts *SlurpOptions) ToDashStr() string {
	options := ""
	for _, types := range opts.Types {
		options += " --types " + types
	}
	if opts.Appearances {
		options += " --appearances"
	}
	options += " " + strings.Join(opts.Addrs, " ")
	options += fmt.Sprintf("%s", "") // auto gen only
	return options
}

func FromRequest(w http.ResponseWriter, r *http.Request) *SlurpOptions {
	opts := &SlurpOptions{}
	for key, value := range r.URL.Query() {
		switch key {
		case "addrs":
			opts.Addrs = append(opts.Addrs, value...)
		case "blocks":
			opts.Blocks = append(opts.Blocks, value...)
		case "types":
			opts.Types = append(opts.Types, value...)
		case "appearances":
			opts.Appearances = true
		}
	}
	opts.Globals = *globals.FromRequest(w, r)

	return opts
}
