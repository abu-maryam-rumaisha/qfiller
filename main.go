// Command qfiller generates a Go function from a string template.
//
//	qfiller -template 'Hello {name}, welcome to {place}!' -func Greet -pkg greet -o greet_gen.go
package main

import (
	"flag"
	"fmt"
	"os"

	"qfiller/generator"
)

func main() {
	tmpl := flag.String("template", `SELECT * FROM {schema}.users u INNER JOIN {schema}.store s on s.user_id = u.id`, "template text, placeholders written as {name} (required)")
	funcName := flag.String("func", "QExample", "name of the generated function")
	pkg := flag.String("pkg", "main", "package name of the generated file")
	out := flag.String("o", "", "output file (default: stdout)")
	flag.Parse()

	if *tmpl == "" {
		fmt.Fprintln(os.Stderr, "qfiller: -template is required")
		flag.Usage()
		os.Exit(2)
	}

	src, err := generator.Generate(*pkg, generator.Spec{FuncName: *funcName, Template: *tmpl})
	if err != nil {
		fmt.Fprintln(os.Stderr, "qfiller:", err)
		os.Exit(1)
	}

	if *out == "" {
		os.Stdout.Write(src)
		return
	}
	if err := os.WriteFile(*out, src, 0o644); err != nil {
		fmt.Fprintln(os.Stderr, "qfiller:", err)
		os.Exit(1)
	}
}
