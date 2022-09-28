package output

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"sync"
	"text/template"

	"github.com/TrueBlocks/trueblocks-core/src/apps/chifra/pkg/logger"
	"github.com/TrueBlocks/trueblocks-core/src/apps/chifra/pkg/rpcClient"
	"github.com/TrueBlocks/trueblocks-core/src/apps/chifra/pkg/types"
)

// OutputOptions allow more granular configuration of output details
type OutputOptions = struct {
	// If set, raw data from the RPC will be printed instead of the model
	ShowRaw bool
	// If set, hidden fields will be printed as well (depends on the format)
	ShowHidden bool
	// If set, the first printed line will be names of the keys in the model
	// (ignored when format is "json")
	ShowKeys bool
	// The format in which to print the output
	Format string
	// How to indent JSON output
	JsonIndent string
	// Meta data to attach to server response
	Meta *rpcClient.MetaData
}

var formatToSeparator = map[string]rune{
	"csv": ',',
	"txt": '\t',
}

// StreamWithTemplate executes a template `tmpl` over Model `model`
func StreamWithTemplate(w io.Writer, model types.Model, tmpl *template.Template) error {
	return tmpl.Execute(w, model.Data)
}

// StreamModel streams a single `Model`
func StreamModel(w io.Writer, model types.Model, options OutputOptions) error {
	if options.Format == "json" || options.Format == "api" {
		v, err := json.MarshalIndent(model.Data, "    ", options.JsonIndent)
		if err != nil {
			return err
		}
		w.Write(v)
		// Add a newline so that the command prompt is not being printed at
		return nil
	}

	// Store map items as strings. All formats other than JSON need string data
	strs := make([]string, 0, len(model.Order))
	for _, key := range model.Order {
		strs = append(strs, fmt.Sprint(model.Data[key]))
	}

	var separator rune
	if len(options.Format) == 1 {
		separator = rune(options.Format[0])
	} else {
		separator = formatToSeparator[options.Format]
	}
	if separator == 0 {
		return fmt.Errorf("unknown format %s", options.Format)
	}
	outputWriter := csv.NewWriter(w)
	outputWriter.Comma = rune(separator)
	if options.ShowKeys {
		outputWriter.Write(model.Order)
	}
	outputWriter.Write(strs)
	// This Flushes for each printed item, but in the exchange the user gets
	// the data printed as it comes
	outputWriter.Flush()

	err := outputWriter.Error()
	if err != nil {
		return err
	}

	return nil
}

// StreamRaw outputs raw `Raw` to `w`
func StreamRaw[Raw types.RawData](w io.Writer, raw *Raw) (err error) {
	bytes, err := json.MarshalIndent(raw, "", "  ")
	if err != nil {
		return
	}
	w.Write(bytes)
	// Add a newline so that the command prompt is not being printed at
	// the same line as the output
	w.Write([]byte("\n"))
	return
}

func writeJsonErrors(w io.Writer, errs []string, options OutputOptions) error {
	marshalled, err := json.MarshalIndent(errs, "  ", options.JsonIndent)
	if err != nil {
		return err
	}
	w.Write(marshalled)
	return nil
}

// StreamMany outputs models or raw data as they are acquired
func StreamMany[Raw types.RawData](
	ctx context.Context,
	w io.Writer,
	// TODO(dszlachta): I renamed this to renderData instead of getData. More accurate
	renderData func(models chan types.Modeler[Raw], errors chan error),
	options OutputOptions,
) error {
	errsToReport := make([]string, 0)
	errsMutex := sync.Mutex{}

	// TODO(dszlachta): let's make channels more obvious. Please rename these to modelChan and errorChan throughout (even into the called function)
	models := make(chan types.Modeler[Raw])
	errors := make(chan error)

	// Check if the current item is the first that we print. If so, we may want to
	// print keys or postpone adding JSON comma between the elements
	first := true
	// Start getting the data
	go func() {
		renderData(models, errors)
		close(models)
		close(errors)
	}()

	// If we are printing JSON, we want to make sure that opening and closing
	// brackets are printed
	if options.Format == "json" {
		w.Write([]byte("{\n  \"data\": [\n    "))
		defer func() {
			w.Write([]byte("\n  ]\n}\n"))
			for _, errMessage := range errsToReport {
				logger.Log(logger.Error, errMessage)
			}
		}()
	}
	// If printing API format, we want to add meta information
	if options.ShowRaw || options.Format == "api" {
		w.Write([]byte("{\n  \"data\": [\n    "))
		defer func() {
			if options.ShowRaw {
				w.Write([]byte("  ]"))
			} else {
				w.Write([]byte("\n  ]"))
			}
			if options.Meta != nil {
				w.Write([]byte(",\n  \"meta\": "))
				b, _ := json.MarshalIndent(options.Meta, "  ", options.JsonIndent)
				w.Write(b)
			}
			if options.Format == "api" && len(errsToReport) > 0 {
				w.Write([]byte(",\n  \"errors\": "))
				err := writeJsonErrors(w, errsToReport, options)
				if err != nil {
					panic(err)
				}
			}
			w.Write([]byte("\n}\n"))
			if options.ShowRaw && options.Format != "api" {
				for _, errMessage := range errsToReport {
					logger.Log(logger.Error, errMessage)
				}
			}
		}()
	}

	defer func() {
		if options.Format == "json" || options.Format == "api" || options.ShowRaw {
			return
		}
		for _, errMessage := range errsToReport {
			logger.Log(logger.Error, errMessage)
		}
	}()

	// If user wants custom format, we have to prepare the template
	customFormat := strings.Contains(options.Format, "{")
	tmpl, err := template.New("").Parse(options.Format)
	if customFormat && err != nil {
		return err
	}

	for {
		select {
		case model, ok := <-models:
			if !ok {
				return nil
			}

			// If the output is JSON and we are printing another item, put `,` in front of it
			if !first && (options.Format == "json" || options.Format == "api") {
				w.Write([]byte(","))
			}
			var err error
			if options.ShowRaw {
				err = StreamRaw(w, model.Raw())
			} else {
				modelValue := model.Model(options.ShowHidden, options.Format)
				if customFormat {
					err = StreamWithTemplate(w, modelValue, tmpl)
				} else {
					err = StreamModel(w, modelValue, OutputOptions{
						ShowKeys:   first && options.ShowKeys,
						Format:     options.Format,
						JsonIndent: "  ",
					})
				}
			}
			if err != nil {
				return err
			}
			first = false

		case err, ok := <-errors:
			if !ok {
				continue
			}
			errsMutex.Lock()
			errsToReport = append(errsToReport, err.Error())
			errsMutex.Unlock()

		case <-ctx.Done():
			return ctx.Err()
		}
	}
}
