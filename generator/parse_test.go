package generator

import (
	"reflect"
	"testing"
)

func TestParse(t *testing.T) {
	lit := func(s string) Part { return Part{Kind: Literal, Value: s} }
	vr := func(s string) Part { return Part{Kind: Variadic, Value: s} }

	tests := []struct {
		name string
		tmpl string
		want []Part
	}{
		{"empty", "", nil},
		{"literal only", "hello", []Part{lit("hello")}},
		{"placeholder only", "{name}", []Part{vr("name")}},
		{"mixed", "Hello {name}, welcome to {place}!", []Part{
			lit("Hello "), vr("name"), lit(", welcome to "), vr("place"), lit("!"),
		}},
		{"adjacent placeholders", "{a}{b}", []Part{vr("a"), vr("b")}},
		{"repeated placeholder", "{x}-{x}", []Part{vr("x"), lit("-"), vr("x")}},
		{"escaped braces", "{{literal}} {v}", []Part{lit("{literal} "), vr("v")}},
		{"unicode", "héllo {wörld}", []Part{lit("héllo "), vr("wörld")}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Parse(tt.tmpl)
			if err != nil {
				t.Fatalf("Parse(%q) error: %v", tt.tmpl, err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Parse(%q)\n got %v\nwant %v", tt.tmpl, got, tt.want)
			}
		})
	}
}

func TestParseErrors(t *testing.T) {
	for _, tmpl := range []string{
		"{",           // unclosed
		"Hello {name", // unclosed
		"}",           // stray close
		"{}",          // empty name
		"{1abc}",      // not an identifier
		"{a b}",       // not an identifier
		"{func}",      // keyword
	} {
		if _, err := Parse(tmpl); err == nil {
			t.Errorf("Parse(%q): expected error", tmpl)
		}
	}
}
