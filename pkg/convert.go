package pkg

import (
	"encoding/json"
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"
)

var matchFirstCap = regexp.MustCompile("(.)([A-Z][a-z]+)")
var matchAllCap = regexp.MustCompile("([a-z0-9])([A-Z])")

func toKebabCase(str string) string {
	snake := matchFirstCap.ReplaceAllString(str, "${1}-${2}")
	snake = matchAllCap.ReplaceAllString(snake, "${1}-${2}")
	return strings.ToLower(snake)
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
	case reflect.Struct, reflect.Pointer:
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
	obj interface{},
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
			return fmt.Errorf("failed to marshal result: %s", err)
		}

		toPrint = append(toPrint, string(buf))
	}

	// if no results, print the object
	if len(toPrint) == 0 {
		buf, _ := json.Marshal(obj)
		fmt.Println(string(buf))
	}

	// print results
	for _, str := range toPrint {
		fmt.Println(str)
	}

	return nil
}
