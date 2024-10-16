package main

var (
	commandTemplate = `// Code generated by gotype2cli; DO NOT EDIT.
package {{.PackageName}}

import (
	{{range .Imports}}"{{.}}"
	{{end}}
	gotype2cli "github.com/pdcalado/gotype2cli/pkg"
)

func make{{.TypeName}}Command(
	fopts ...func(*gotype2cli.CreateCommandOptions),
) (*cobra.Command, error) {
	opts := gotype2cli.CreateCommandOptions{
		TypeName: "{{.TypeName}}",
		MethodArgs: map[string][]string{
			{{range .FunctionData}}"{{.Name}}": []string{ {{range .Args}}"{{.}}",{{end}} },
			{{end}}
		},
		MethodDocs: map[string]string{
			{{range .FunctionData}}{{if .Docs }}"{{.Name}}": {{.Docs | printf "%q" }},{{end}}
			{{end}}
		},
		Constructors: map[string]reflect.Value{
			{{range .FunctionData}}{{if eq .Kind Constructor}}"{{.Name}}": reflect.ValueOf({{.Name}}),{{end}}{{end}}
		},
		ReceiverPrint: {{.ReceiverPrint}},
	}

	for _, f := range fopts {
		f(&opts)
	}

	return gotype2cli.CreateCommand(
		reflect.TypeOf({{.TypeName}}{}),
		&opts,
	)
}
`
)
