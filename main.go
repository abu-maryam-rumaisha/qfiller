// Command qfiller generates Go functions from string templates.
//
//	qfiller -template 'Hello {name}, welcome to {place}!' -func Greet -pkg greet -o greet_gen.go
//	qfiller -src ./store -o ./store/queries_gen.go
package main

import (
	"flag"
	"fmt"
	"os"

	"qfiller/generator"
)

func main() {
	tmpl := flag.String("template", `SELECT * FROM {schema}.users u INNER JOIN {schema}.store s on s.user_id = u.id`, "template text, placeholders written as {name} (ignored with -src)")
	funcName := flag.String("func", "QExample", "name of the generated function (ignored with -src)")
	srcPath := flag.String("src", "", "Go file or directory to read //qfiller: comments from")
	pkg := flag.String("pkg", "main", "package name of the generated file (default with -src: the scanned package)")
	out := flag.String("o", "", "output file (default: stdout)")
	flag.Parse()

	pkgSet := false
	flag.Visit(func(f *flag.Flag) {
		if f.Name == "pkg" {
			pkgSet = true
		}
	})

	var specs []generator.Spec
	if *srcPath != "" {
		srcPkg, found, err := generator.Scan(*srcPath)
		if err != nil {
			fail(err)
		}
		if len(found) == 0 {
			fail(fmt.Errorf("no //qfiller: directives found in %s", *srcPath))
		}
		if !pkgSet {
			*pkg = srcPkg
		}
		specs = found
	} else {
		if *tmpl == "" {
			fmt.Fprintln(os.Stderr, "qfiller: -template or -src is required")
			flag.Usage()
			os.Exit(2)
		}
		specs = []generator.Spec{{FuncName: *funcName, Template: *tmpl}}
	}

	src, err := generator.Generate(*pkg, specs...)
	if err != nil {
		fail(err)
	}

	if *out == "" {
		os.Stdout.Write(src)
		return
	}
	if err := os.WriteFile(*out, src, 0o644); err != nil {
		fail(err)
	}
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, "qfiller:", err)
	os.Exit(1)
}
