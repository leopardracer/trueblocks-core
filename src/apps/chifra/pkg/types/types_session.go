package types

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/TrueBlocks/trueblocks-core/src/apps/chifra/pkg/file"
	"github.com/TrueBlocks/trueblocks-core/src/apps/chifra/pkg/utils"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// Session stores ephemeral things such as last window position,
// last view, and recent file list.
type Session struct {
	LastChain  string            `json:"lastChain"`
	LastFile   string            `json:"lastFile"`
	LastFolder string            `json:"lastFolder"`
	LastRoute  string            `json:"lastRoute"`
	LastSub    map[string]string `json:"lastSub"`
	Window     Window            `json:"window"`
	Wizard     Wizard            `json:"wizard"`
	Toggles    Toggles           `json:"toggles"`
	Chain      string            `json:"-"`
}

func (s Session) String() string {
	bytes, _ := json.Marshal(s)
	return string(bytes)
}

func (s *Session) Model(chain, format string, verbose bool, extraOpts map[string]any) Model {
	var model = map[string]any{}
	var order = []string{}

	model = map[string]any{
		"lastChain":  s.LastChain,
		"lastFile":   s.LastFile,
		"lastFolder": s.LastFolder,
		"lastRoute":  s.LastRoute,
		"window":     s.Window,
		"wizard":     s.Wizard,
		"toggles":    s.Toggles,
	}
	order = []string{
		"lastChain",
		"lastFile",
		"lastFolder",
		"lastRoute",
		"window",
		"wizard",
		"toggles",
	}

	return Model{model, order}
}

func (s *Session) FinishUnmarshal() {
	// EXISTING_CODE
	// EXISTING_CODE
}

func (s *Session) ShallowCopy() Session {
	return *s
}

const theTitle = "Browse by TrueBlocks"

var defLayout = Layout{
	Header: true,
	Menu:   true,
	Help:   true,
	Footer: true,
}

var defHeader = Headers{
	Project:   false,
	History:   true,
	Monitors:  false,
	Names:     false,
	Abis:      false,
	Indexes:   false,
	Manifests: false,
	Status:    true,
	Settings:  true,
}

var defDaemons = Daemons{
	Freshen: true,
}

var defaultSession = Session{
	LastChain: "mainnet",
	LastFile:  "Untitled.tbx",
	LastRoute: "/wizard",
	LastSub:   map[string]string{"/history": "0xf503017d7baf7fbc0fff7492b751025c6a78179b"},
	Window: Window{
		X:      0,
		Y:      0,
		Width:  0,
		Height: 0,
		Title:  theTitle,
	},
	Wizard: Wizard{State: Welcome},
	Toggles: Toggles{
		Layout:  defLayout,
		Headers: defHeader,
		Daemons: defDaemons,
	},
}

// Save saves the session to the configuration folder.
func (s *Session) Save() error {
	if fn, err := utils.GetConfigFn("browse", "session.json"); err != nil {
		return err
	} else {
		if contents, _ := json.MarshalIndent(s, "", "  "); len(contents) > 0 {
			_ = file.StringToAsciiFile(fn, string(contents))
		}
		return nil
	}
}

var ErrLoadingSession = errors.New("error loading session")

// Load loads the session from the configuration folder. If the file contains
// data, we return true. False otherwise.
func (s *Session) Load() error {
	loaded := false
	defer func() {
		if !loaded {
			*s = defaultSession
		} else {
			// Ensure a valid file (if for example the user edited it)
			if s.Wizard.State == Okay && s.LastRoute == "/wizard" {
				s.LastRoute = "/"
			}
			if s.LastChain == "" {
				s.LastChain = "mainnet"
			}
			if s.LastFile == "" {
				s.LastFile = "Untitled.tbx"
			}
		}
		_ = s.Save() // creates the session file if it doesn't already exist
	}()

	fn, err := utils.GetConfigFn("browse", "session.json")
	if err != nil {
		return fmt.Errorf("%w: %v", ErrLoadingSession, err)
	}

	contents := file.AsciiFileToString(fn)
	if len(contents) == 0 {
		// This is not an error (the default session will be used)
		return nil
	}

	if err = json.Unmarshal([]byte(contents), s); err != nil {
		return fmt.Errorf("%w: %v", ErrLoadingSession, err)
	}

	loaded = true
	return nil
}

func (s *Session) SetRoute(route, subRoute string) {
	s.LastRoute = route
	if len(subRoute) > 0 {
		s.LastSub[route] = subRoute
	}
	_ = s.Save()
}

var ErrScreenNotFound = errors.New("screen not found")

// CleanWindowSize ensures a valid window size. (If the app has never run before
// or the session fails to load its width or height will be zero.) This function
// always returns a valid window size, but it may also return an error.
func (s *Session) CleanWindowSize(ctx context.Context) (Window, error) {
	// Any window size other than 0,0 is already okay.
	if s.Window.Width != 0 && s.Window.Height != 0 {
		return s.Window, nil
	}

	ret := Window{X: 30, Y: 30, Width: 1024, Height: 768}
	defer func() {
		_ = s.Save()
	}()

	if screens, err := runtime.ScreenGetAll(ctx); err != nil {
		return ret, fmt.Errorf("error getting screens %w", err)

	} else {
		var fullScreen *Window = nil
		for _, screen := range screens {
			if screen.IsCurrent || screen.IsPrimary {
				fullScreen = &Window{
					Width:  screen.Size.Width,
					Height: screen.Size.Height,
				}
				break
			}
		}
		if fullScreen != nil {
			// We found the screen, so we can set a reasonable window size.
			s.Window.X = fullScreen.Width / 6
			s.Window.Y = fullScreen.Width / 6
			s.Window.Width = (5 * fullScreen.Width) / 6
			s.Window.Height = (5 * fullScreen.Width) / 6
		}
	}
	return s.Window, nil
}

type Layout struct {
	Header bool `json:"header"`
	Menu   bool `json:"menu"`
	Help   bool `json:"help"`
	Footer bool `json:"footer"`
}

type Headers struct {
	Project   bool `json:"project"`
	History   bool `json:"history"`
	Monitors  bool `json:"monitors"`
	Names     bool `json:"names"`
	Abis      bool `json:"abis"`
	Indexes   bool `json:"indexes"`
	Manifests bool `json:"manifests"`
	Status    bool `json:"status"`
	Settings  bool `json:"settings"`
}

type Daemons struct {
	Freshen bool `json:"freshen"`
	Scraper bool `json:"scraper"`
	Ipfs    bool `json:"ipfs"`
}

type Toggles struct {
	Layout  Layout  `json:"layout"`
	Headers Headers `json:"headers"`
	Daemons Daemons `json:"daemons"`
}

func (t *Toggles) IsOn(which string) bool {
	if which == "" {
		which = "project"
	}
	switch which {
	case "header":
		return t.Layout.Header
	case "menu":
		return t.Layout.Menu
	case "help":
		return t.Layout.Help
	case "footer":
		return t.Layout.Footer
	case "project":
		return t.Headers.Project
	case "history":
		return t.Headers.History
	case "monitors":
		return t.Headers.Monitors
	case "names":
		return t.Headers.Names
	case "abis":
		return t.Headers.Abis
	case "indexes":
		return t.Headers.Indexes
	case "manifests":
		return t.Headers.Manifests
	case "status":
		return t.Headers.Status
	case "settings":
		return t.Headers.Settings
	case "freshen":
		return t.Daemons.Freshen
	case "scraper":
		return t.Daemons.Scraper
	case "ipfs":
		return t.Daemons.Ipfs
	}
	return false
}

func (t *Toggles) SetState(which string, onOff bool) {
	if which == "" {
		which = "project"
	}
	switch which {
	case "header":
		t.Layout.Header = onOff
	case "menu":
		t.Layout.Menu = onOff
	case "help":
		t.Layout.Help = onOff
	case "footer":
		t.Layout.Footer = onOff
	case "project":
		t.Headers.Project = onOff
	case "history":
		t.Headers.History = onOff
	case "monitors":
		t.Headers.Monitors = onOff
	case "names":
		t.Headers.Names = onOff
	case "abis":
		t.Headers.Abis = onOff
	case "indexes":
		t.Headers.Indexes = onOff
	case "manifests":
		t.Headers.Manifests = onOff
	case "status":
		t.Headers.Status = onOff
	case "settings":
		t.Headers.Settings = onOff
	case "freshen":
		t.Daemons.Freshen = onOff
	case "scraper":
		t.Daemons.Scraper = onOff
	case "ipfs":
		t.Daemons.Ipfs = onOff
	}
}

// Window stores the last position and title of the window
type Window struct {
	X      int    `json:"x"`
	Y      int    `json:"y"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
	Title  string `json:"title"`
}

func (w *Window) String() string {
	bytes, _ := json.Marshal(w)
	return string(bytes)
}

type Wizard struct {
	State WizState `json:"state"`
}

var stateOrder = []WizState{
	Welcome,
	Error,
	TomlOkay,
	RpcOkay,
	BloomsOkay,
	IndexOkay,
	Okay,
}

func (w *Wizard) Step(step WizStep) {
	switch step {
	case Reset:
		w.State = Error
	case Previous:
		if w.State == TomlOkay {
			w.State = Welcome
		} else {
			for i := range stateOrder {
				if stateOrder[i] == w.State && i > 0 {
					w.State = stateOrder[i-1]
					break
				}
			}
		}
	case Next:
		if w.State == Welcome {
			w.State = TomlOkay
		} else {
			for i := range stateOrder {
				if stateOrder[i] == w.State && i < len(stateOrder)-1 {
					w.State = stateOrder[i+1]
					break
				}
			}
		}
	case Finish:
		w.State = Okay
	}
}

type WizState string

const (
	Welcome    WizState = "welcome"
	TomlOkay   WizState = "tomlOkay"
	RpcOkay    WizState = "rpcOkay"
	BloomsOkay WizState = "bloomsOkay"
	IndexOkay  WizState = "indexOkay"
	Error      WizState = "error"
	Okay       WizState = "okay"
)

// String returns the string representation of the WizState.
func (s WizState) String() string {
	return string(s)
}

// AllStates - all possible WizStates for the frontend codegen
var AllStates = []struct {
	Value  WizState `json:"value"`
	TSName string   `json:"tsName"`
}{
	{Welcome, "WELCOME"},
	{TomlOkay, "TOMLOKAY"},
	{RpcOkay, "RPCOKAY"},
	{BloomsOkay, "BLOOMSOKAY"},
	{IndexOkay, "INDEXOKAY"},
	{Error, "ERROR"},
	{Okay, "OKAY"},
}

type WizStep string

const (
	Reset    WizStep = "Reset"
	Previous WizStep = "Previous"
	Next     WizStep = "Next"
	Finish   WizStep = "Finish"
)

// String returns the string representation of the Step.
func (s WizStep) String() string {
	return string(s)
}

// AllSteps - all possible steps for the frontend codegen
var AllSteps = []struct {
	Value  WizStep `json:"value"`
	TSName string  `json:"tsName"`
}{
	{Reset, "RESET"},
	{Previous, "PREVIOUS"},
	{Next, "NEXT"},
	{Finish, "FINISH"},
}
