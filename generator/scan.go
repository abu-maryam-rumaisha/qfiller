package generator

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// directive marks a comment that declares a template, as in
//
//	//qfiller:QUsers SELECT * FROM {schema}.users
const directive = "qfiller:"

// Scan reads Go source files and collects the templates declared in their
// comments. It returns the package name of the sources and one Spec per
// directive, in source order.
//
// Each path is a .go file or a directory. For a directory, its .go files are
// read in name order, skipping _test.go files and generated files (so the
// output of a previous run is never read back in).
//
// A line comment directive is written as
//
//	//qfiller:FuncName optional template text
//	// more template text
//
// The template is the text after the function name, plus every following
// line comment in the same comment group (up to the next directive), joined
// with newlines. One space after // is stripped from each following line.
//
// A block comment directive is written as
//
//	/*qfiller:FuncName
//	SELECT *
//	FROM {schema}.users
//	*/
//
// and its template is everything after the function name, with surrounding
// newlines and trailing blank space removed.
func Scan(paths ...string) (pkg string, specs []Spec, err error) {
	files, err := expandPaths(paths)
	if err != nil {
		return "", nil, err
	}
	if len(files) == 0 {
		return "", nil, fmt.Errorf("no Go files found")
	}

	fset := token.NewFileSet()
	seen := map[string]token.Position{}
	for _, path := range files {
		f, err := parser.ParseFile(fset, path, nil, parser.ParseComments|parser.SkipObjectResolution)
		if err != nil {
			return "", nil, err
		}
		if pkg == "" {
			pkg = f.Name.Name
		} else if f.Name.Name != pkg {
			return "", nil, fmt.Errorf("%s: package %s, expected %s", path, f.Name.Name, pkg)
		}

		found, err := scanFile(fset, f)
		if err != nil {
			return "", nil, err
		}
		for _, d := range found {
			if prev, ok := seen[d.FuncName]; ok {
				return "", nil, fmt.Errorf("%s: duplicate function name %q (first declared at %s)", d.pos, d.FuncName, prev)
			}
			seen[d.FuncName] = d.pos
			specs = append(specs, d.Spec)
		}
	}
	return pkg, specs, nil
}

type found struct {
	Spec
	pos token.Position
}

// scanFile returns the directives in f, checking each template with Parse so
// that errors point at the source position.
func scanFile(fset *token.FileSet, f *ast.File) ([]found, error) {
	var out []found
	for _, group := range f.Comments {
		var cur *found
		var lines []string
		finish := func() error {
			if cur == nil {
				return nil
			}
			cur.Template = strings.Join(lines, "\n")
			if err := checkDirective(cur); err != nil {
				return err
			}
			out = append(out, *cur)
			cur, lines = nil, nil
			return nil
		}

		for _, c := range group.List {
			if body, ok := strings.CutPrefix(c.Text, "/*"+directive); ok {
				if err := finish(); err != nil {
					return nil, err
				}
				name, tmpl := cutName(strings.TrimSuffix(body, "*/"))
				tmpl = strings.TrimLeft(tmpl, "\n")
				tmpl = strings.TrimRight(tmpl, " \t\r\n")
				d := found{Spec: Spec{FuncName: name, Template: tmpl}, pos: fset.Position(c.Pos())}
				if err := checkDirective(&d); err != nil {
					return nil, err
				}
				out = append(out, d)
				continue
			}

			text, isLine := strings.CutPrefix(c.Text, "//")
			if !isLine {
				// An ordinary block comment ends the current template.
				if err := finish(); err != nil {
					return nil, err
				}
				continue
			}
			if body, ok := strings.CutPrefix(text, directive); ok {
				if err := finish(); err != nil {
					return nil, err
				}
				name, tmpl := cutName(body)
				cur = &found{Spec: Spec{FuncName: name}, pos: fset.Position(c.Pos())}
				if tmpl != "" {
					lines = append(lines, tmpl)
				}
				continue
			}
			if cur != nil {
				lines = append(lines, strings.TrimPrefix(text, " "))
			}
		}
		if err := finish(); err != nil {
			return nil, err
		}
	}
	return out, nil
}

// cutName splits a directive body into the function name and the template
// text after it. One separating space or tab is removed; a newline is kept.
func cutName(body string) (name, rest string) {
	i := strings.IndexAny(body, " \t\r\n")
	if i < 0 {
		return body, ""
	}
	name, rest = body[:i], body[i:]
	if rest[0] == ' ' || rest[0] == '\t' {
		rest = rest[1:]
	}
	return name, rest
}

func checkDirective(d *found) error {
	if d.FuncName == "" {
		return fmt.Errorf("%s: qfiller directive has no function name", d.pos)
	}
	if !token.IsIdentifier(d.FuncName) || token.IsKeyword(d.FuncName) {
		return fmt.Errorf("%s: invalid function name %q", d.pos, d.FuncName)
	}
	if strings.TrimSpace(d.Template) == "" {
		// Also catches gofmt moving a directive to the end of a doc comment.
		return fmt.Errorf("%s: %s has no template (put the directive in its own comment, not a doc comment)", d.pos, d.FuncName)
	}
	if _, err := Parse(d.Template); err != nil {
		return fmt.Errorf("%s: %s: %w", d.pos, d.FuncName, err)
	}
	return nil
}

func expandPaths(paths []string) ([]string, error) {
	var files []string
	for _, p := range paths {
		info, err := os.Stat(p)
		if err != nil {
			return nil, err
		}
		if !info.IsDir() {
			files = append(files, p)
			continue
		}
		entries, err := os.ReadDir(p)
		if err != nil {
			return nil, err
		}
		var names []string
		for _, e := range entries {
			n := e.Name()
			if e.IsDir() || !strings.HasSuffix(n, ".go") || strings.HasSuffix(n, "_test.go") {
				continue
			}
			path := filepath.Join(p, n)
			generated, err := isGenerated(path)
			if err != nil {
				return nil, err
			}
			if !generated {
				names = append(names, path)
			}
		}
		sort.Strings(names)
		files = append(files, names...)
	}
	return files, nil
}

func isGenerated(path string) (bool, error) {
	f, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.PackageClauseOnly|parser.ParseComments)
	if err != nil {
		return false, err
	}
	return ast.IsGenerated(f), nil
}
