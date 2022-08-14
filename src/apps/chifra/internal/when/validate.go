// Copyright 2021 The TrueBlocks Authors. All rights reserved.
// Use of this source code is governed by a license that can
// be found in the LICENSE file.

package whenPkg

import (
	"errors"

	"github.com/TrueBlocks/trueblocks-core/src/apps/chifra/pkg/utils"
	"github.com/TrueBlocks/trueblocks-core/src/apps/chifra/pkg/validate"
)

func (opts *WhenOptions) validateWhen() error {
	opts.testLog()

	if opts.BadFlag != nil {
		return opts.BadFlag
	}

	if len(opts.Blocks) == 0 {
		if !opts.List && !opts.Timestamps {
			return validate.Usage("Please supply one or more block identifiers or one or more dates.")
		}

	} else {
		if opts.List {
			return validate.Usage("Please supply either {0} or the {1} option.", "block identifiers", "--list")

		} else if opts.Timestamps {
			if opts.Truncate != utils.NOPOS {
				return validate.Usage("Please supply {0} with the {1} option.", "one or more block identifiers", "--reset")
			}
		}
	}

	if opts.Timestamps {
		if opts.List {
			return validate.Usage("Please choose only one of {0}.", "--timestamps or --list")
		}
		if opts.Deep && !opts.Check {
			return validate.Usage("The {0} option is only available with the {1} option.", "--deep", "--timestamps --check")
		}

	} else {
		if opts.Check {
			return validate.Usage("The {0} option is only available with the {1} option.", "--check", "--timestamps")
		}
		if opts.Deep {
			return validate.Usage("The {0} option is only available with the {1} option.", "--deep", "--timestamps --check")
		}
		if opts.Count {
			return validate.Usage("The {0} option is only available with the {1} option.", "--count", "--timestamps")
		}
		if opts.Truncate != utils.NOPOS {
			return validate.Usage("The {0} option is only available with the {1} option.", "--drop", "--timestamps")
		}
	}

	if opts.Globals.TestMode && opts.Timestamps && !opts.Check && !opts.Count {
		return validate.Usage("--timestamp option not tested in testMode")
	}

	err := validate.ValidateIdentifiers(
		opts.Globals.Chain,
		opts.Blocks,
		validate.ValidBlockIdWithRangeAndDate,
		1,
		&opts.BlockIds,
	)
	if err != nil {
		if invalidLiteral, ok := err.(*validate.InvalidIdentifierLiteralError); ok {
			return invalidLiteral
		}

		if errors.Is(err, validate.ErrTooManyRanges) {
			return validate.Usage("Specify only a single block range at a time.")
		}

		return err
	}

	return opts.Globals.Validate()
}
