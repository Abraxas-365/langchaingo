package main

import (
	"fmt"
	"go/ast"
	"go/format"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/tools/go/packages"
)

type FunctionInfo struct {
	FunctionBody *ast.FuncDecl
	FunctionText string
	GptResume    string
	CalledFuncs  []*ast.FuncDecl
}

var functionMap map[string]FunctionInfo
var functionNameMap map[string]*ast.FuncDecl

// collectRelatedNodes returns a list of function call expressions found in the given node.
func collectRelatedNodes(pkg *packages.Package, node ast.Node) []*ast.CallExpr {
	var relatedNodes []*ast.CallExpr

	ast.Inspect(node, func(childNode ast.Node) bool {
		callExpr, ok := childNode.(*ast.CallExpr)
		if ok {
			relatedNodes = append(relatedNodes, callExpr)
		}
		return true
	})

	return relatedNodes
}

// populateFunctionMaps processes the package's AST to find all function declarations,
// populates the functionMap and functionNameMap, and links the called functions.
func populateFunctionMaps(pkg *packages.Package) {
	for _, file := range pkg.Syntax {
		ast.Inspect(file, func(node ast.Node) bool {
			fn, ok := node.(*ast.FuncDecl)
			if ok {
				functionKey := fmt.Sprintf("%s:%s", pkg.Fset.Position(fn.Pos()).Filename, fn.Name.Name)

				//Traer las funciones que son llamadas desde esa funcion
				relatedNodes := collectRelatedNodes(pkg, fn)
				calledFuncs := make([]*ast.FuncDecl, 0, len(relatedNodes))

				for _, callExpr := range relatedNodes {
					var funName string
					switch fun := callExpr.Fun.(type) {
					case *ast.Ident:
						funName = fun.Name
					case *ast.SelectorExpr:
						funName = fun.Sel.Name
					}

					if funName != "" {
						funcBody, ok := functionNameMap[funName]
						if ok {
							calledFuncs = append(calledFuncs, funcBody)
						}
					}
				}

				var fnBody strings.Builder
				if err := format.Node(&fnBody, pkg.Fset, fn); err != nil {
					panic(err)
				}



				functionMap[functionKey] = FunctionInfo{
					FunctionBody: fn,
					FunctionText: fnBody.String(),
					CalledFuncs:  calledFuncs,
				}

				functionNameMap[fn.Name.Name] = fn
			}
			return true
		})
	}
}

// processGoFile processes a single Go file and populates functionMap and functionNameMap.
func processGoFile(path string) error {
	dir := filepath.Dir(path)

	cfg := &packages.Config{
		Mode:  packages.LoadAllSyntax,
		Tests: false,
		Dir:   dir,
	}

	pkgs, err := packages.Load(cfg, ".")
	if err != nil {
		return err
	}

	if len(pkgs) != 1 {
		return fmt.Errorf("expected a single package, but got %d packages", len(pkgs))
	}

	pkg := pkgs[0]
	populateFunctionMaps(pkg)

	return nil
}

func main() {
	targetDir := "./test"
	functionMap = make(map[string]FunctionInfo)
	functionNameMap = make(map[string]*ast.FuncDecl)

	err := filepath.Walk(targetDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(path, ".go") {
			err := processGoFile(path)
			if err != nil {
				fmt.Printf("Error: %v\n", err)
				return err
			}
		}
		return nil
	})

	if err != nil {
		fmt.Printf("Error: %v\n", err)
	}

	// Print the function map
	for key, value := range functionMap {
		fmt.Printf("Key: %s\n", key)
		fmt.Println(value.FunctionText)
		fmt.Printf("Called Functions:\n")
		for _, calledFunc := range value.CalledFuncs {
			fmt.Printf("\t%s\n", calledFunc.Name.Name)
		}
	}
}
