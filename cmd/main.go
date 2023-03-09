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

	var typesList []string
	var typesDocs []string

	for _, filename := range pkg.GoFiles {
		node, err := parser.ParseFile(fset, filename, nil, parser.ParseComments)
		if err != nil {
			log.Fatalf("failed to parse file %s: %s", filename, err)
		}

		funcDocs := make(map[string]*FunctionDocs)

		ast.Inspect(node, func(n ast.Node) bool {
			funcDoc := inspect(targetTypeName, n)
			if funcDoc != nil {
				funcDocs[funcDoc.Name] = funcDoc
			}
			return true
		})

		funcNames := make([]string, 0, len(funcDocs))
		for funcName := range funcDocs {
			funcNames = append(funcNames, funcName)
		}

		sort.Strings(funcNames)

		for _, funcName := range funcNames {
			args := funcDocs[funcName].Args
			stringedArgs := make([]string, 0, len(args))
			for _, arg := range args {
				stringedArgs = append(stringedArgs, fmt.Sprintf(`"%s"`, arg))
			}

			typesList = append(typesList, fmt.Sprintf("\"%s\": []string{%s},\n", funcName, strings.Join(stringedArgs, ", ")))

			doc := funcDocs[funcName].Docs
			if doc != "" {
				typesDocs = append(typesDocs, fmt.Sprintf("\"%s\": `%s`,\n", funcName, doc))
			}
		}
	}

	tmpl, err := template.New("manifest").Parse(commandTemplate)
	if err != nil {
		log.Fatal(err)
	}

	data := struct {
		PackageName       string
		Imports           []string
		MethodArgsVarName string
		MethodDocsVarName string
		MethodArgsList    string
		MethodDocsList    string
	}{
		PackageName:       pkg.Name,
		Imports:           []string{"fmt", "os", "log"},
		MethodArgsVarName: fmt.Sprintf("methodArgs%s", targetTypeName),
		MethodDocsVarName: fmt.Sprintf("methodDocs%s", targetTypeName),
		MethodArgsList:    strings.Join(typesList, ""),
		MethodDocsList:    strings.Join(typesDocs, ""),
	}

	tmpl.Execute(os.Stdout, data)
}

type FunctionDocs struct {
	Name string
	Args []string
	Docs string
}

// returns map of func args and map of doc args
func inspect(typeName string, n ast.Node) (funcDoc *FunctionDocs) {

	switch x := n.(type) {
	case *ast.FuncDecl:
		if x.Recv == nil ||
			len(x.Recv.List) != 1 ||
			x.Recv.List[0].Type == nil {
			return
		}

		receiver, ok := x.Recv.List[0].Type.(*ast.StarExpr)
		if !ok {
			return
		}

		recvType, ok := receiver.X.(*ast.Ident)
		if !ok {
			return
		}

		if recvType.Name != typeName {
			return
		}

		if x.Type.Params == nil {
			return
		}

		funcDoc = &FunctionDocs{
			Name: x.Name.Name,
		}

		for _, param := range x.Type.Params.List {
			funcDoc.Args = append(funcDoc.Args, param.Names[0].Name)
		}

		if x.Doc == nil {
			return
		}

		comments := make([]string, 0, len(x.Doc.List))
		for _, comment := range x.Doc.List {
			text := comment.Text
			if strings.HasPrefix(text, "/") {
				text = strings.TrimLeft(text, "/ ")
			}
			comments = append(comments, text)
		}
		funcDoc.Docs = strings.Join(comments, "\n")
	}

	return
}
