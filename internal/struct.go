package golang

import (
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/sqlc-dev/plugin-sdk-go/plugin"
	"github.com/sqlc-dev/sqlc-gen-go/internal/opts"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

type Struct struct {
	Table   *plugin.Identifier
	Name    string
	Package string
	Fields  []Field
	Comment string
}

func (s Struct) Type() string {
	if s.Package != "" {
		return s.Package + "." + s.Name
	}
	return s.Name
}

func StructName(name string, options *opts.Options) string {
	if rename := options.Rename[name]; rename != "" {
		return rename
	}
	caser := cases.Title(language.English)
	out := ""
	name = strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) {
			return r
		}
		if unicode.IsDigit(r) {
			return r
		}
		return rune('_')
	}, name)

	for _, p := range strings.Split(name, "_") {
		if _, found := options.InitialismsMap[p]; found {
			out += strings.ToUpper(p)
		} else {
			out += caser.String(p)
		}
	}

	// If a name has a digit as its first char, prepand an underscore to make it a valid Go name.
	r, _ := utf8.DecodeRuneInString(out)
	if unicode.IsDigit(r) {
		return "_" + out
	} else {
		return out
	}
}
