package cmd

import (
	"bufio"
	"log"
	"os"
)

// testFlags implements the plugin.FlagSet interface.
type testFlags struct {
	bools       map[string]bool
	ints        map[string]int
	floats      map[string]float64
	strings     map[string]string
	args        []string
	stringLists map[string][]string
}

func (testFlags) ExtraUsage() string { return "" }

func (testFlags) AddExtraUsage(eu string) {}

func (f testFlags) Bool(s string, d bool, c string) *bool {
	if b, ok := f.bools[s]; ok {
		return &b
	}
	return &d
}

func (f testFlags) Int(s string, d int, c string) *int {
	if i, ok := f.ints[s]; ok {
		return &i
	}
	return &d
}

func (f testFlags) Float64(s string, d float64, c string) *float64 {
	if g, ok := f.floats[s]; ok {
		return &g
	}
	return &d
}

func (f testFlags) String(s, d, c string) *string {
	if t, ok := f.strings[s]; ok {
		return &t
	}
	return &d
}

func (f testFlags) StringList(s, d, c string) *[]*string {
	if t, ok := f.stringLists[s]; ok {
		// convert slice of strings to slice of string pointers before returning.
		tp := make([]*string, len(t))
		for i, v := range t {
			tp[i] = &v
		}
		return &tp
	}
	return &[]*string{}
}

func (f testFlags) Parse(func()) []string {
	return f.args
}

func baseFlags() testFlags {
	return testFlags{
		bools: map[string]bool{
			"text": true,
		},
		ints: map[string]int{},
		floats: map[string]float64{
			"nodefraction": 0.05,
			"edgefraction": 0.01,
		},
		strings: map[string]string{},
	}
}

type UI struct {
	r *bufio.Reader
}

func (ui *UI) ReadLine(prompt string) (string, error) {
	os.Stdout.WriteString(prompt)
	return ui.r.ReadString('\n')
}

func (ui *UI) Print(args ...interface{}) {
	log.Print(args...)
}

func (ui *UI) PrintErr(args ...interface{}) {
	log.Print(args...)
}

func (ui *UI) IsTerminal() bool {
	return false
}

func (ui *UI) WantBrowser() bool {
	return false
}

func (ui *UI) SetAutoComplete(func(string) string) {
}
