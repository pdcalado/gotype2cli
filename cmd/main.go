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
	types         = flag.String("type", "", "list of types separated by comma (required)")
	doWrite       = flag.Bool("w", false, "write result to (source) file instead of stdout")
	receiverPrint = flag.Bool("receiver-print", true, "print receiver when method returns only error or void")
)

func Usage() {
	_, _ = fmt.Fprintf(os.Stderr, "gotype2cli generates Go code to create CLI command from your Go types.\n")
	_, _ = fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
	_, _ = fmt.Fprintf(os.Stderr, "\tgotype2cli [flags] -types Types [<directory>]\n")
	_, _ = fmt.Fprintf(os.Stderr, "\ndirectory: \".\" if unspecified\n\n")
	_, _ = fmt.Fprintf(os.Stderr, "Flags:\n")
	flag.PrintDefaults()
}

func main() {
	log.SetPrefix("gotype2cli: ")
	flag.Usage = Usage
	flag.Parse()
	if len(*types) == 0 {
		flag.Usage()
		os.Exit(1)
	}

	directory := "."
	if len(flag.Args()) > 0 {
		directory = flag.Args()[0]
	}

	typeList := strings.Split(*types, ",")

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
		PackageName   string
		Imports       []string
		TypeName      string
		FunctionData  []*FunctionData
		ReceiverPrint bool
	}{
		PackageName:   pkg.Name,
		Imports:       imports,
		TypeName:      targetTypeName,
		FunctionData:  listFunctionData,
		ReceiverPrint: *receiverPrint,
	}

	writer := os.Stdout
	if *doWrite {
		filename := fmt.Sprintf("%s_gotype2cli.go", strings.ToLower(targetTypeName))
		writer, err = os.Create(filename)
		if err != nil {
			log.Fatalf("failed to create file %s: %s", filename, err)
		}
	}

	err = tmpl.Execute(writer, data)
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
		for _, name := range param.Names {
			funcData.Args = append(funcData.Args, name.Name)
		}
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

	receiver := x.Recv.List[0].Type

	if receiver == nil {
		return false
	}

	name := ""

	switch o := receiver.(type) {
	case *ast.Ident:
		name = o.Name
	case *ast.StarExpr:
		recvType, ok := o.X.(*ast.Ident)
		if !ok {
			return false
		}

		name = recvType.Name
	default:
		return false
	}

	if name != typeName {
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
