package pkg

import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"strings"

	"github.com/mattn/go-isatty"
	"github.com/spf13/cobra"
)

type CreateCommandOptions struct {
	// TypeName is the name of the type to generate a command for
	TypeName string
	// CommandName is the name of the command to generate
	// (if empty, we use the type's name converted to kebab-case)
	CommandName string
	// Constructors is a map of functions that return objects of target type
	Constructors map[string]reflect.Value
	// MethodArgs is a map of method names to list of argument names
	MethodArgs map[string][]string
	// MethodDocs is a map of method names to documentation strings
	MethodDocs map[string]string
}

func CreateCommand(
	targetType reflect.Type,
	options *CreateCommandOptions,
) (*cobra.Command, error) {

	commandName := options.CommandName
	if options.CommandName == "" {
		commandName = toKebabCase(options.TypeName)
	}

	methodArgs := options.MethodArgs
	methodDocs := options.MethodDocs

	typeCmd := &cobra.Command{
		Use:   commandName,
		Short: "",
	}

	pointerType := reflect.PtrTo(targetType)

	for i := 0; i < pointerType.NumMethod(); i++ {
		method := pointerType.Method(i)

		var callArgTypes []reflect.Type
		for j := 1; j < method.Type.NumIn(); j++ {
			callArgTypes = append(callArgTypes, method.Type.In(j))
		}

		argNames, ok := methodArgs[method.Name]
		if !ok {
			return nil, fmt.Errorf("missing arg names for method '%s'", method.Name)
		}

		if len(argNames) != len(callArgTypes) {
			return nil, fmt.Errorf("wrong number of arg names for method '%s'", method.Name)
		}

		description, hasDoc := methodDocs[method.Name]
		if !hasDoc {
			return nil, fmt.Errorf("missing doc for method '%s', add comment and generate again", method.Name)
		}

		subCmd := &cobra.Command{
			Use:   toKebabCase(method.Name) + " " + strings.Join(argNames, " "),
			Short: description,
			Long:  description,
			Args:  cobra.ExactArgs(len(argNames)),
			RunE:  methodCommandRunner(targetType, method.Func, callArgTypes),
		}
		typeCmd.AddCommand(subCmd)
	}

	// add constructors
	for name, constructor := range options.Constructors {
		constructorType := constructor.Type()

		var callArgTypes []reflect.Type
		for j := 1; j < constructorType.NumIn(); j++ {
			callArgTypes = append(callArgTypes, constructorType.In(j))
		}

		argNames, ok := methodArgs[name]
		if !ok {
			return nil, fmt.Errorf("missing arg names for method '%s'", name)
		}

		if len(argNames) != len(callArgTypes) {
			return nil, fmt.Errorf("wrong number of arg names for method '%s'", name)
		}

		description, hasDoc := methodDocs[name]
		if !hasDoc {
			return nil, fmt.Errorf("missing doc for method '%s', add comment and generate again", name)
		}

		subCmd := &cobra.Command{
			Use:   toKebabCase(name) + " " + strings.Join(argNames, " "),
			Short: description,
			Long:  description,
			Args:  cobra.ExactArgs(len(argNames)),
			RunE:  constructorCommandRunner(targetType, constructor, callArgTypes),
		}
		typeCmd.AddCommand(subCmd)
	}

	return typeCmd, nil
}

func methodCommandRunner(
	targetType reflect.Type,
	method reflect.Value,
	callArgTypes []reflect.Type,
) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		callArgs, err := convertInputs(args, callArgTypes)
		if err != nil {
			return fmt.Errorf("failed to convert args: %s", err)
		}

		object := reflect.New(targetType).Interface()

		// check if data is being piped from stdin
		if !isatty.IsTerminal(os.Stdin.Fd()) &&
			!isatty.IsCygwinTerminal(os.Stdin.Fd()) {

			dec := json.NewDecoder(os.Stdin)
			err := dec.Decode(object)
			if err != nil {
				return fmt.Errorf("failed to read from stdin: %s", err)
			}
		}

		allArgs := append([]reflect.Value{reflect.ValueOf(object)}, callArgs...)

		result := method.Call(allArgs)

		return outputResults(object, result)
	}
}

func constructorCommandRunner(
	targetType reflect.Type,
	method reflect.Value,
	callArgTypes []reflect.Type,
) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		callArgs, err := convertInputs(args, callArgTypes)
		if err != nil {
			return fmt.Errorf("failed to convert args: %s", err)
		}

		object := reflect.New(targetType).Interface()

		result := method.Call(callArgs)

		return outputResults(object, result)
	}
}
