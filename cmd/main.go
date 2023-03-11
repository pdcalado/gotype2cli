package main

import (
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"log"
	"os"
	"sort"
	"strings"
	"text/template"

	"golang.org/x/tools/go/packages"
)

var (
	types = flag.String("type", "", "list of types separated by comma (required)")
)

func Usage() {
	_, _ = fmt.Fprintf(os.Stderr, "gotype2cli generates Go code to create CLI command from your Go types.\n")
	_, _ = fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
	_, _ = fmt.Fprintf(os.Stderr, "\tgotype2cli [flags] -types Types <directory>\n")
	_, _ = fmt.Fprintf(os.Stderr, "Flags:\n")
	flag.PrintDefaults()
}

func main() {
	log.SetPrefix("gotype2cli: ")
	flag.Usage = Usage
	flag.Parse()
	if len(*types) == 0 || len(flag.Args()) == 0 {
		flag.Usage()
		os.Exit(1)
	}

	typeList := strings.Split(*types, ",")

	directory := flag.Args()[0]

	loadAllSyntax := packages.NeedName | packages.NeedFiles | packages.NeedCompiledGoFiles | packages.NeedImports | packages.NeedTypes | packages.NeedTypesSizes | packages.NeedSyntax | packages.NeedTypesInfo

	cfg := &packages.Config{
		Mode:  loadAllSyntax,
		Tests: false,
		Dir:   directory,
	}
	pkgs, err := packages.Load(cfg, "")
	if err != nil {
		log.Fatal(err)
	}

	if len(pkgs) != 1 {
		log.Fatalf("error: %d packages found, expected 1", len(pkgs))
	}

	pkg := pkgs[0]

	fset := token.NewFileSet()

	targetTypeName := typeList[0]

	mapFunctionData := make(map[string]*FunctionData)

	for _, filename := range pkg.GoFiles {
		node, err := parser.ParseFile(fset, filename, nil, parser.ParseComments)
		if err != nil {
			log.Fatalf("failed to parse file %s: %s", filename, err)
		}

		ast.Inspect(node, func(n ast.Node) bool {
			funcDoc := inspect(targetTypeName, n)
			if funcDoc != nil {
				mapFunctionData[funcDoc.Name] = funcDoc
			}
			return true
		})
	}

	// convert map to list with sorted function names
	funcNames := make([]string, 0, len(mapFunctionData))
	for funcName := range mapFunctionData {
		funcNames = append(funcNames, funcName)
	}

	sort.Strings(funcNames)

	listFunctionData := make([]*FunctionData, 0, len(mapFunctionData))
	for _, funcName := range funcNames {
		listFunctionData = append(listFunctionData, mapFunctionData[funcName])
	}

	funcMap := template.FuncMap{
		"Constructor": Constructor.get,
	}

	tmpl, err := template.New("template").Funcs(funcMap).Parse(commandTemplate)
	if err != nil {
		log.Fatal(err)
	}

	imports := []string{
		"reflect",
		"github.com/spf13/cobra",
	}

	data := struct {
		PackageName  string
		Imports      []string
		TypeName     string
		FunctionData []*FunctionData
	}{
		PackageName:  pkg.Name,
		Imports:      imports,
		TypeName:     targetTypeName,
		FunctionData: listFunctionData,
	}

	err = tmpl.Execute(os.Stdout, data)
	if err != nil {
		log.Fatalf("failed to execute template: %s", err)
	}
}

type FunctionKind int

const (
	Unrelated FunctionKind = iota
	Receiver
	Constructor
)

func (k FunctionKind) get() FunctionKind { return k }

type FunctionData struct {
	Name string
	Args []string
	Docs string
	Kind FunctionKind
}

// returns function args and docs
func inspect(typeName string, n ast.Node) (funcData *FunctionData) {
	x, isFuncDecl := n.(*ast.FuncDecl)
	if !isFuncDecl {
		return
	}

	if x.Type.Params == nil {
		return
	}

	var kind FunctionKind
	switch {
	case isReceiverMethod(typeName, x):
		kind = Receiver
	case isConstructorMethod(typeName, x):
		kind = Constructor
	default:
		return
	}

	funcData = &FunctionData{
		Name: x.Name.Name,
		Kind: kind,
	}

	for _, param := range x.Type.Params.List {
		funcData.Args = append(funcData.Args, param.Names[0].Name)
	}

	if x.Doc == nil {
		return
	}

	comments := make([]string, 0, len(x.Doc.List))
	for _, comment := range x.Doc.List {
		text := strings.TrimLeft(comment.Text, "/ ")
		comments = append(comments, text)
	}
	funcData.Docs = strings.Join(comments, "\n")

	return
}

func isReceiverMethod(typeName string, x *ast.FuncDecl) bool {
	if x.Recv == nil ||
		len(x.Recv.List) != 1 ||
		x.Recv.List[0].Type == nil {
		return false
	}

	receiver, ok := x.Recv.List[0].Type.(*ast.StarExpr)
	if !ok {
		return false
	}

	recvType, ok := receiver.X.(*ast.Ident)
	if !ok {
		return false
	}

	if recvType.Name != typeName {
		return false
	}

	return true
}

func isConstructorMethod(typeName string, x *ast.FuncDecl) bool {
	if x.Recv != nil && len(x.Recv.List) > 0 {
		return false
	}

	if x.Type.Results == nil || len(x.Type.Results.List) == 0 {
		return false
	}

	for _, resultField := range x.Type.Results.List {
		switch result := resultField.Type.(type) {
		case *ast.StarExpr:
			x, ok := result.X.(*ast.Ident)
			if !ok {
				return false
			}
			if x.Name == typeName {
				return true
			}
		case *ast.Ident:
			if result.Name == typeName {
				return true
			}
		}
	}

	return true
}
