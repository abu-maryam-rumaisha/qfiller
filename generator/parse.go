// Package generator turns string templates into Go source code built with go/ast.
package generator

import (
	"fmt"
	"go/token"
	"strings"
)

// PartKind tells whether a part is fixed text or a runtime value.
type PartKind int

const (
	// Literal is fixed text known at generation time.
	Literal PartKind = iota
	// Variadic is a value passed in as a parameter of the generated function.
	Variadic
)

func (k PartKind) String() string {
	switch k {
	case Literal:
		return "Literal"
	case Variadic:
		return "Variadic"
	}
	return fmt.Sprintf("PartKind(%d)", int(k))
}

// Part is one piece of a parsed template. For a Literal, Value is the text;
// for a Variadic, Value is the placeholder name.
type Part struct {
	Kind  PartKind
	Value string
}

// Parse splits a template into ordered literal and variadic parts.
//
// Placeholders are written as {name}, where name must be a valid Go
// identifier that is not a keyword. Use {{ and }} for literal braces.
// Adjacent literal text is merged into a single part.
func Parse(tmpl string) ([]Part, error) {
	var parts []Part
	var lit strings.Builder

	flush := func() {
		if lit.Len() > 0 {
			parts = append(parts, Part{Kind: Literal, Value: lit.String()})
			lit.Reset()
		}
	}

	for i := 0; i < len(tmpl); i++ {
		c := tmpl[i]
		switch c {
		case '{':
			if i+1 < len(tmpl) && tmpl[i+1] == '{' {
				lit.WriteByte('{')
				i++
				continue
			}
			end := strings.IndexByte(tmpl[i+1:], '}')
			if end < 0 {
				return nil, fmt.Errorf("unclosed placeholder at offset %d", i)
			}
			name := tmpl[i+1 : i+1+end]
			if err := checkName(name); err != nil {
				return nil, fmt.Errorf("placeholder at offset %d: %w", i, err)
			}
			flush()
			parts = append(parts, Part{Kind: Variadic, Value: name})
			i += end + 1
		case '}':
			if i+1 < len(tmpl) && tmpl[i+1] == '}' {
				lit.WriteByte('}')
				i++
				continue
			}
			return nil, fmt.Errorf("unexpected '}' at offset %d (use '}}' for a literal brace)", i)
		default:
			lit.WriteByte(c)
		}
	}
	flush()
	return parts, nil
}

func checkName(name string) error {
	switch {
	case name == "":
		return fmt.Errorf("empty placeholder name")
	case token.IsKeyword(name):
		return fmt.Errorf("placeholder name %q is a Go keyword", name)
	case !token.IsIdentifier(name):
		return fmt.Errorf("placeholder name %q is not a valid Go identifier", name)
	}
	return nil
}
