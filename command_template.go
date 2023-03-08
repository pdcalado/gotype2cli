package main

var (
	commandTemplate = `
package {{.PackageName}}

import (
	{{range .Imports}}"{{.}}"
	{{end}}
)

var {{.MethodArgsVarName}} = map[string][]string{
	{{.MethodArgsList | printf "%s"}}
}

var {{.MethodDocsVarName}} = map[string]string{
	{{.MethodDocsList}}
}

var matchFirstCap = regexp.MustCompile("(.)([A-Z][a-z]+)")
var matchAllCap = regexp.MustCompile("([a-z0-9])([A-Z])")

func ToKebabCase(str string) string {
	snake := matchFirstCap.ReplaceAllString(str, "${1}-${2}")
	snake = matchAllCap.ReplaceAllString(snake, "${1}-${2}")
	return strings.ToLower(snake)
}

func create{{.TypeName}}Command() *cobra.Command {

	{{.TypeNameCamel}}Cmd := &cobra.Command{
		Use:   ToKebabCase("{{.TypeName}}"),
		Short: "",
	}

	// use reflect to list methods of type Metadata
	targetType := reflect.TypeOf(&{{.TypeName}}{})
	for i := 0; i < targetType.NumMethod(); i++ {
		method := targetType.Method(i)

		var callArgTypes []reflect.Type

		for j := 0; j < method.Type.NumIn(); j++ {
			if j == 0 {
				continue
			}

			ty := method.Type.In(j)

			callArgTypes = append(callArgTypes, ty)
		}

		argNames, ok := {{.MethodArgsVarName}}[method.Name]
		if !ok {
			log.Fatal(fmt.Errorf("missing arg names for method '%s', try running 'make qgenerate'", method.Name))
		}

		if len(argNames) != len(callArgTypes) {
			log.Fatal(fmt.Errorf("wrong number of arg names for method '%s'", method.Name))
		}

		// create a command for each method
		use := fmt.Sprintf("%s %s", ToKebabCase(method.Name), strings.Join(argNames, " "))

		longDescription, hasDoc := {{.MethodDocsVarName}}[method.Name]
		if !hasDoc {
			log.Fatal(fmt.Errorf("missing doc for method '%s', add comment and generate again", method.Name))
		}

		cmd := &cobra.Command{
			Use:  use,
			Long: longDescription,
			Args: cobra.ExactArgs(len(argNames)),
			RunE: func(cmd *cobra.Command, args []string) error {
				callArgs, err := metadataConvertInputs(args[1:], callArgTypes)
				if err != nil {
					return fmt.Errorf("failed to convert args: %s", err)
				}

				metadata, err := metadataReadFromArg(args[0])
				if err != nil {
					return err
				}

				allArgs := append([]reflect.Value{reflect.ValueOf(metadata)}, callArgs...)

				result := method.Func.Call(allArgs)

				return metadataOutputResults(metadata, result)
			},
		}
		{{.TypeNameCamel}}Cmd.AddCommand(cmd)
	}

	return {{.TypeNameCamel}}Cmd
}

func convertInputs(
	args []string,
	types []reflect.Type,
) ([]reflect.Value, error) {

	values := make([]reflect.Value, len(args))

	for i, arg := range args {
		value, err := convertInput(arg, types[i])
		if err != nil {
			return nil, err
		}
		values[i] = value
	}

	return values, nil
}

func convertInput(arg string, ty reflect.Type) (reflect.Value, error) {
	value := reflect.New(ty).Elem()

	switch ty.Kind() {
	case reflect.String:
		value.SetString(arg)
	case reflect.Int:
		i, err := strconv.ParseInt(arg, 10, 64)
		if err != nil {
			return reflect.Value{}, err
		}
		value.SetInt(i)
	case reflect.Bool:
		b, err := strconv.ParseBool(arg)
		if err != nil {
			return reflect.Value{}, err
		}
		value.SetBool(b)
	case reflect.Slice:
		if ty.Elem() == reflect.TypeOf(byte(0)) {
			value.SetBytes([]byte(arg))
			break
		}

		obj := reflect.New(ty).Interface()
		err := json.Unmarshal([]byte(arg), obj)
		if err != nil {
			return reflect.Value{}, err
		}
		value.Set(reflect.ValueOf(obj).Elem())
	case reflect.Struct:
		obj := reflect.New(ty).Interface()
		err := json.Unmarshal([]byte(arg), obj)
		if err != nil {
			return reflect.Value{}, err
		}
		value.Set(reflect.ValueOf(obj).Elem())
	default:
		panic("not supported")
	}

	return value, nil
}

func outputResults(
	{{.TypeNameCamel}} *{{.TypeName}},
	results []reflect.Value,
) error {
	var toPrint []string

	for _, result := range results {
		// check if a result is an error
		if result.Type() == reflect.TypeOf((*error)(nil)).Elem() {
			if !result.IsNil() {
				return result.Interface().(error)
			}
			continue
		}

		// convert result to json
		buf, err := json.Marshal(result.Interface())
		if err != nil {
			return errors.New("failed to marshal result: %s", err)
		}

		toPrint = append(toPrint, string(buf))
	}

	// if no results, print the metadata
	if len(toPrint) == 0 {
		buf, _ := json.Marshal(metadata)
		fmt.Println(string(buf))
	}

	// print results
	for _, str := range toPrint {
		fmt.Println(str)
	}

	return nil
}

func readFromStdin() (*{{.TypeName}}, error) {
	var {{.TypeNameCamel}} {{.TypeName}}

	dec := json.NewDecoder(os.Stdin)
	err := dec.Decode(&{{.TypeNameCamel}})
	return &{{.TypeNameCamel}}, err
}
`
)
