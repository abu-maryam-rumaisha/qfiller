package bench

import (
	"strings"
	"testing"

	"github.com/abu-maryam-rumaisha/qfiller/generator"
)

// schema is a variable so the compiler cannot fold either call into a constant.
var schema = "tenant_42"

// sink keeps results alive so the calls are not optimized away.
var sink string

// usersTemplate reads the QUsers template from the //qfiller: directive in
// bench.go, so ReplaceAll works on exactly the text QUsers was generated from.
func usersTemplate(tb testing.TB) string {
	tb.Helper()
	_, specs, err := generator.Scan(".")
	if err != nil {
		tb.Fatal(err)
	}
	for _, s := range specs {
		if s.FuncName == "QUsers" {
			return s.Template
		}
	}
	tb.Fatal("no //qfiller:QUsers directive")
	return ""
}

func TestSameResult(t *testing.T) {
	got := QUsers(schema)
	want := strings.ReplaceAll(usersTemplate(t), "{schema}", schema)
	if got != want {
		t.Errorf("QUsers = %q, ReplaceAll = %q", got, want)
	}
}

func BenchmarkReplaceAll(b *testing.B) {
	tmpl := usersTemplate(b)
	b.ReportAllocs()
	for b.Loop() {
		sink = strings.ReplaceAll(tmpl, "{schema}", schema)
	}
}

func BenchmarkGenerated(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		sink = QUsers(schema)
	}
}
