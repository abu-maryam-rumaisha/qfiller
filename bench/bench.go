// Package bench compares qfiller's generated code with strings.ReplaceAll.
package bench

//go:generate go run .. -src . -o query_gen.go

//qfiller:QUsers SELECT * FROM {schema}.users u INNER JOIN {schema}.store s on s.user_id = u.id
