package main

var (
	commandTemplate = `package {{.PackageName}}

import (
	{{range .Imports}}"{{.}}"
	{{end}}
	gotype2cli "github.com/pdcalado/gotype2cli/pkg"
)

func make{{.TypeName}}Command() (*cobra.Command, error) {
	opts := gotype2cli.CreateCommandOptions{
		TypeName: "{{.TypeName}}",
		MethodArgs: map[string][]string{
			{{.MethodArgsList}}
		},
		MethodDocs: map[string]string{
			{{.MethodDocsList}}
		},
		Constructors: map[string]reflect.Value{
			{{.ConstructorsList}}
		},
	}

	return gotype2cli.CreateCommand(
		reflect.TypeOf({{.TypeName}}{}),
		&opts,
	)
}
`
)
