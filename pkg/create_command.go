package pkg

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
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
	// FailOnMissingDocs generator fails if a method has no documentation
	FailOnMissingDocs bool
	// ReceiverPrint prints the receiver object for methods that return only
	// an error or nothing
	ReceiverPrint bool
	// DefaultConstructor must return a new object of the target type.
	// If nil, the zero value of the target type is used.
	// Make sure a pointer to the target type is returned.
	DefaultConstructor func() interface{}
}

func CreateCommand(
	targetType reflect.Type,
	options *CreateCommandOptions,
) (*cobra.Command, error) {

	defaultConstructor := func() interface{} {
		return reflect.New(targetType).Interface()
	}

	if options.DefaultConstructor != nil {
		defaultConstructor = options.DefaultConstructor
	}

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

	pointerType := reflect.PointerTo(targetType)

	for i := 0; i < pointerType.NumMethod(); i++ {
		method := pointerType.Method(i)

		hasContext := false

		var callArgTypes []reflect.Type
		for j := 1; j < method.Type.NumIn(); j++ {
			if j == 1 {
				iface := reflect.New(method.Type.In(j)).Interface()
				_, isCtx := iface.(*context.Context)
				if isCtx {
					hasContext = true
					continue
				}
			}

			callArgTypes = append(callArgTypes, method.Type.In(j))
		}

		argNames, ok := methodArgs[method.Name]
		if !ok {
			return nil, fmt.Errorf("missing arg names for method '%s'", method.Name)
		}

		if hasContext {
			argNames = argNames[1:]
		}

		if len(argNames) != len(callArgTypes) {
			return nil, fmt.Errorf("wrong number of arg names for method '%s'", method.Name)
		}

		description, hasDoc := methodDocs[method.Name]
		if !hasDoc && options.FailOnMissingDocs {
			return nil, fmt.Errorf("missing doc for method '%s', add comment and generate again", method.Name)
		}

		taggedArgNames := make([]string, len(argNames))
		for i, argName := range argNames {
			taggedArgNames[i] = fmt.Sprintf("<%s>", argName)
		}

		runner := methodCommandRunner{
			targetType:         targetType,
			method:             method.Func,
			callArgTypes:       callArgTypes,
			hasContext:         hasContext,
			receiverPrint:      options.ReceiverPrint,
			defaultConstructor: defaultConstructor,
		}

		subCmd := &cobra.Command{
			Use:   toKebabCase(method.Name) + " " + strings.Join(taggedArgNames, " "),
			Short: description,
			Long:  description,
			Args:  cobra.ExactArgs(len(argNames)),
			RunE:  runner.RunE,
		}
		typeCmd.AddCommand(subCmd)
	}

	// add constructors
	for name, constructor := range options.Constructors {
		constructorType := constructor.Type()

		var callArgTypes []reflect.Type
		for j := 0; j < constructorType.NumIn(); j++ {
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

		taggedArgNames := make([]string, len(argNames))
		for i, argName := range argNames {
			taggedArgNames[i] = fmt.Sprintf("<%s>", argName)
		}

		runner := constructorCommandRunner{
			targetType:    targetType,
			method:        constructor,
			callArgTypes:  callArgTypes,
			receiverPrint: options.ReceiverPrint,
		}

		subCmd := &cobra.Command{
			Use:   toKebabCase(name) + " " + strings.Join(taggedArgNames, " "),
			Short: description,
			Long:  description,
			Args:  cobra.ExactArgs(len(argNames)),
			RunE:  runner.RunE,
		}
		typeCmd.AddCommand(subCmd)
	}

	return typeCmd, nil
}

type methodCommandRunner struct {
	targetType         reflect.Type
	method             reflect.Value
	callArgTypes       []reflect.Type
	hasContext         bool
	receiverPrint      bool
	defaultConstructor func() interface{}
}

func (r methodCommandRunner) RunE(cmd *cobra.Command, args []string) error {
	callArgs, err := convertInputs(args, r.callArgTypes)
	if err != nil {
		return fmt.Errorf("failed to convert args: %s", err)
	}

	if r.hasContext {
		callArgs = append([]reflect.Value{reflect.ValueOf(cmd.Context())}, callArgs...)
	}

	object := r.defaultConstructor()

	// check if data is being piped from stdin
	if !isatty.IsTerminal(os.Stdin.Fd()) &&
		!isatty.IsCygwinTerminal(os.Stdin.Fd()) {

		dec := json.NewDecoder(os.Stdin)
		err := dec.Decode(object)
		if err != nil && err != io.EOF { // ignore EOF
			return fmt.Errorf("failed to read from stdin: %s", err)
		}
	}

	allArgs := append([]reflect.Value{reflect.ValueOf(object)}, callArgs...)

	var result []reflect.Value
	if r.method.Type().IsVariadic() {
		result = r.method.CallSlice(allArgs)
	} else {
		result = r.method.Call(allArgs)
	}

	printed, err := outputResults(cmd, result)
	if err != nil {
		return err
	}

	if printed {
		return nil
	}

	if r.receiverPrint {
		buf, _ := json.Marshal(object)
		_, err := cmd.OutOrStdout().Write(buf)
		return err
	}

	return nil
}

type constructorCommandRunner struct {
	targetType    reflect.Type
	method        reflect.Value
	callArgTypes  []reflect.Type
	receiverPrint bool
}

func (r constructorCommandRunner) RunE(cmd *cobra.Command, args []string) error {
	callArgs, err := convertInputs(args, r.callArgTypes)
	if err != nil {
		return fmt.Errorf("failed to convert args: %s", err)
	}

	object := reflect.New(r.targetType).Interface()

	var result []reflect.Value
	if r.method.Type().IsVariadic() {
		result = r.method.CallSlice(callArgs)
	} else {
		result = r.method.Call(callArgs)
	}

	printed, err := outputResults(cmd, result)
	if err != nil {
		return err
	}

	if printed {
		return nil
	}

	buf, _ := json.Marshal(object)
	_, err = cmd.OutOrStdout().Write(buf)
	return err
}
