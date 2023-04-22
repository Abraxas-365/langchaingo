package main

import (
	"fmt"
	"go/ast"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/tools/go/packages"
)

type FunctionInfo struct {
	FunctionBody *ast.FuncDecl
	CalledFuncs  []*ast.FuncDecl
}

var functionMap map[string]FunctionInfo
var functionNameMap map[string]*ast.FuncDecl

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

func processPackage(pkg *packages.Package) {
	for _, file := range pkg.Syntax {
		ast.Inspect(file, func(node ast.Node) bool {
			fn, ok := node.(*ast.FuncDecl)
			if ok {
				functionKey := fmt.Sprintf("%s:%s", pkg.Fset.Position(fn.Pos()).Filename, fn.Name.Name)

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

				functionMap[functionKey] = FunctionInfo{
					FunctionBody: fn,
					CalledFuncs:  calledFuncs,
				}

				functionNameMap[fn.Name.Name] = fn
			}
			return true
		})
	}
}

func main() {
	targetDir := "./test"
	functionMap = make(map[string]FunctionInfo)
	functionNameMap = make(map[string]*ast.FuncDecl)

	err := filepath.Walk(targetDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			return err
		}
		if !info.IsDir() && strings.HasSuffix(path, ".go") {
			dir := filepath.Dir(path)

			cfg := &packages.Config{
				Mode:  packages.LoadAllSyntax,
				Tests: false,
				Dir:   dir,
			}

			pkgs, err := packages.Load(cfg, ".")
			if err != nil {
				fmt.Printf("Error: %v\n", err)
				return err
			}

			if len(pkgs) != 1 {
				fmt.Printf("Expected a single package, but got %d packages\n", len(pkgs))
				return nil
			}

			pkg := pkgs[0]
			processPackage(pkg)
		}
		return nil
	})

	if err != nil {
		fmt.Printf("Error: %v\n", err)
	}

	// Print the function map
	for key, value := range functionMap {
		fmt.Printf("Key: %s\n", key)
		ast.Print(nil, value.FunctionBody)
		fmt.Printf("Called Functions:\n")
		for _, calledFunc := range value.CalledFuncs {
			fmt.Printf("\t%s\n", calledFunc.Name.Name)
		}
	}
}
