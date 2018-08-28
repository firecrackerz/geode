package ast

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strings"

	"path/filepath"

	"github.com/geode-lang/geode/pkg/lexer"
	"github.com/geode-lang/geode/pkg/util"
	"github.com/geode-lang/geode/pkg/util/log"
	"github.com/geode-lang/llvm/ir"
	"github.com/geode-lang/llvm/ir/types"
)

// Program is a wrapper for information used
// in codegen and dependency resolution
type Program struct {
	Scope           *Scope
	Compiler        *Compiler
	Module          *ir.Module
	ParsedFiles     []string
	Packages        map[string]*Package
	Package         *Package // the currently active package
	CLinkages       []string
	Entry           string
	TargetTripple   string
	TypePrecidences map[types.Type]int
	Functions       map[string]*FunctionNode
	Classes         map[string]*ClassNode
	Initializations []*GlobalVariableDeclNode
}

// NewProgram creates a program and returns a pointer to it
func NewProgram() *Program {
	p := &Program{}
	p.Scope = NewScope()
	p.Scope.InjectPrimitives()
	p.Compiler = &Compiler{}
	p.Module = ir.NewModule()
	p.Packages = make(map[string]*Package)
	p.Initializations = make([]*GlobalVariableDeclNode, 0)

	p.TypePrecidences = make(map[types.Type]int)
	p.TypePrecidences[types.I1] = 1
	p.TypePrecidences[types.I8] = 2
	p.TypePrecidences[types.I16] = 3
	p.TypePrecidences[types.I32] = 4
	p.TypePrecidences[types.I64] = 5
	p.TypePrecidences[types.Double] = 11
	p.TypePrecidences[types.NewPointer(types.I8)] = 0
	p.TypePrecidences[types.Void] = 0

	return p
}

// ParsePath parses from some some path and handles
// everything required to get a final compiled program from some
// basic source location
func (p *Program) ParsePath(dir string) {

	// Determine if the path is a directory or not.
	// if isDir, _ := PathIsDir(dir); !isDir {
	// 	// The path isn't a directory, so we just pull the base of the file
	// 	dir = filepath.Dir(dir)
	// }

	dir = ReduceToDir(dir)

	absEntry, err := filepath.Abs(dir)

	if err != nil {
		log.Fatal("Error with parsing entry location\n")
	}

	files, err := p.ParseDir(absEntry)
	if err != nil {
		fmt.Println(err)
		log.Fatal("Error parsing folder for geode source files\n")
	}

	for _, file := range files {
		p.ParseFile(file)
	}
}

// CanParse helps decide whether or not to parse a file based on previously parsed files
func (p *Program) CanParse(file string) bool {
	for _, parsed := range p.ParsedFiles {
		if parsed == file {
			return false
		}
	}
	return true
}

// ParseDir parses a directory for all package information
func (p *Program) ParseDir(path string) ([]string, error) {
	fd, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer fd.Close()

	list, err := fd.Readdir(-1)
	if err != nil {
		return nil, err
	}

	files := make([]string, 0, len(list))

	// pkgs = make(map[string]*ast.Package)
	for _, file := range list {
		if strings.HasSuffix(file.Name(), ".g") {
			filename := filepath.Join(path, file.Name())
			if p.CanParse(filename) {
				files = append(files, filename)
			}
		}
	}

	return files, nil

}

// ParseFile will parse the contents of the file at some path into a Package
func (p *Program) ParseFile(path string) {
	p.ParsedFiles = append(p.ParsedFiles, path)
	src, err := lexer.NewSourcefile(path)
	if err != nil {
		fmt.Println(err)
		log.Fatal("Error creating Sourcefile context for file at %q\n", path)
	}
	src.LoadFile(path)

	tokens := lexer.Lex(src)

	nodes := make([]Node, 0)

	for node := range Parse(tokens) {
		nodes = append(nodes, node)
	}

	name, err := NamespaceFromNodes(nodes)
	if err != nil {
		log.Fatal("Unable to decide on namespace for file %q", filepath.Clean(path))
	}

	newPkg := NewPackage(name, p)
	newPkg.Program = p
	newPkg.Files[path] = src
	newPkg.Nodes = nodes

	_, found := p.Packages[path]
	if !found {
		p.Packages[path] = newPkg
	}

	for node := range FilterNodes(newPkg.Nodes, nodeDependency) {
		base := filepath.Dir(path)
		dep := node.(DependencyNode)
		for _, depPath := range dep.Paths {
			if dep.CLinkage {
				p.CLinkages = append(p.CLinkages, ResolveDepPath(base, depPath))
			} else {
				newPkg.DependencyPaths = append(newPkg.DependencyPaths, ReduceToDir(ResolveDepPath(base, depPath)))
				p.ParseDep(base, depPath)

			}
		}

	}

}

// ParseDep will parse any dependency relative to the current base
func (p *Program) ParseDep(base, path string) {
	depPath := ResolveDepPath(base, path)
	if p.CanParse(depPath) {
		p.ParsePath(depPath)
	}
}

// ReduceToDir takes a path and reduces it down into its directory
func ReduceToDir(path string) string {
	if isDir, err := PathIsDir(path); !isDir || err != nil {
		path = filepath.Dir(path)
	}
	return path
}

// RegisterFunction takes a name and a function and registers it in the
// program's storage
func (p *Program) RegisterFunction(name string, fn FunctionNode) {
	p.Functions[name] = &fn
}

// Congeal sets the programs module to one with nodes filled out
func (p *Program) Congeal() *ir.Module {
	p.Module = ir.NewModule()

	nodes := make([]*PackagedNode, 0)

	p.Functions = make(map[string]*FunctionNode)
	p.Classes = make(map[string]*ClassNode)
	p.Compiler = NewCompiler(p)

	for _, pkg := range p.Packages {
		for _, node := range pkg.Nodes {

			if fn, is := node.(FunctionNode); is {
				name := fmt.Sprintf("%s:%s", pkg.Name, fn.Name)
				if fn.Name.String() == "main" || pkg.Name == "_runtime" {
					name = fn.Name.String()
				}
				fn.Package = pkg
				p.Functions[name] = &fn
			}

			if cls, is := node.(ClassNode); is {
				name := fmt.Sprintf("%s:%s", pkg.Name, cls.Name)
				if pkg.Name == "_runtime" {
					name = cls.Name
				}
				p.Classes[name] = &cls
			}
			nodes = append(nodes, PackageNode(node, pkg, p))
		}
	}

	for node := range FilterPackagedNodes(nodes, nodeClass) {
		node.SetupContext()
		node.Node.(ClassNode).Declare(p)
	}

	// Codegen the types/classes
	for node := range FilterPackagedNodes(nodes, nodeClass) {
		node.SetupContext()
		err := node.Node.(ClassNode).VerifyCorrectness(p)
		util.EatError(err)
		node.Node.(ClassNode).Codegen(p)
	}

	for node := range FilterPackagedNodes(nodes, nodeGlobalDecl) {
		node.SetupContext()
		node.Node.(GlobalVariableDeclNode).Declare(p)
	}

	// err := sem.Check(p.Module)
	// if err != nil {
	// 	log.Fatal("Semantic Check Error: %s\n", err)
	// }

	return p.Module
}

// CastPrecidence takes some type and returns the precidence
func (p *Program) CastPrecidence(t types.Type) int {
	if val, exists := p.TypePrecidences[t]; exists {
		return val
	}
	return -1
}

// FunctionCompilationOptions contains options for function compilation
type FunctionCompilationOptions struct {
	ArgTypes []types.Type
}

// RegisterGlobalVariableInitialization -
func (p *Program) RegisterGlobalVariableInitialization(node *GlobalVariableDeclNode) {
	p.Initializations = append(p.Initializations, node)
}

// CompileFunction takes a funciton node, detects if it is already compiled or not
// if it isnt compiled, it will codegen, otherwise it will return the compiled one
func (p *Program) CompileFunction(name string, options FunctionCompilationOptions) *ir.Function {

	// Save the program state
	previousPackage := p.Package
	previousScope := p.Scope
	previousCompiler := p.Compiler.Copy()

	node, exists := p.Functions[name]
	if !exists {
		return nil
	}

	// Prime the program's new state before compiling a function
	p.Package = node.Package
	p.Scope = p.Scope.GetRoot()
	p.Scope.PackageName = p.Package.Name
	p.Scope = p.Scope.SpawnChild()
	p.Compiler = NewCompiler(p)

	_, rawTypes := node.Arguments(p.Scope)

	if node.Variants == nil {
		node.Variants = make(map[string]*ir.Function)
	}

	correctTypes := make([]types.Type, 0, len(rawTypes))

	if options.ArgTypes != nil && !node.Variadic {

		for i, expected := range rawTypes {

			nodeParamType := node.Args[i].Type
			given := options.ArgTypes[i]
			unknown := nodeParamType.Unknown

			if (expected != nil && given != nil) && !types.Equal(expected, given) && !typesAreLooselyEqual(given, expected) && !unknown {
				node.Args[i].SyntaxError()
				log.Fatal("Incorrect type passed into function %s. Given: %q, Expected: %q\n", node.Name, given, expected)
			}

			if unknown {
				// Handling unknown types's scope definition on call
				p.Scope.RegisterType(node.Args[i].Type.Name, given, 0)
				correctTypes = append(correctTypes, given)
			} else {
				correctTypes = append(correctTypes, expected)
			}
		}
	}

	var compiledVal *ir.Function

	if node.Nomangle {
		node.NameCache = node.Name.Value
	} else {
		node.NameCache = node.MangledName(p, correctTypes)
	}

	if f, found := node.Variants[node.NameCache]; found {
		compiledVal = f
	} else {
		// if !node.External && node.Name.String() != "main" {
		// 	node.Name.Value = node.MangledName(p, correctTypes) // MangleFunctionName(fmt.Sprintf("%s:%s", node.Package.Name, node.Name.String()), types)
		// }

		node.Variants[node.NameCache] = node.Declare(p) // Declare first to allow recursive calls
		node.Compiled = true
		if !node.External {
			node.Variants[node.NameCache] = node.Codegen(p).(*ir.Function)
		}

		compiledVal = node.Variants[node.NameCache]
	}

	p.Package = previousPackage
	p.Scope = previousScope
	p.Compiler = previousCompiler
	return compiledVal
}

// Emit will emit the package as IR to a file then build it into an object file for further usage.
// This function returns the path to the object file
func (p *Program) Emit(buildDir string) string {
	outPathBase, _ := filepath.Abs(p.Entry)

	outPathBase = path.Join(buildDir, outPathBase)
	extension := filepath.Ext(outPathBase)
	outPathBase = outPathBase[0 : len(outPathBase)-len(extension)]

	baseDir := filepath.Dir(outPathBase)

	os.MkdirAll(baseDir, os.ModePerm)

	llvmFileName := fmt.Sprintf("%s.ll", outPathBase)

	ir := p.String()

	writeErr := ioutil.WriteFile(llvmFileName, []byte(ir), 0666)
	if writeErr != nil {
		panic(writeErr)
	}

	return llvmFileName
}

// String will get the LLVM IR from the package's compiler
func (p *Program) String() string {
	ir := &bytes.Buffer{}
	// We need to build up the IR that will be emitted
	// so we can track this information later on.
	fmt.Fprintf(ir, "target datalayout = %q\n", "e-m:o-i64:64-f80:128-n8:16:32:64-S128")
	fmt.Fprintf(ir, "target triple = %q\n", p.TargetTripple)

	// Append the module information
	fmt.Fprintf(ir, "\n%s", p.Compiler.Module.String())

	return ir.String()
}

// ResolveDepPath returns the absolute location to a dependency
func ResolveDepPath(base, filename string) string {
	if strings.HasPrefix(filename, "std:") {
		filename = strings.Replace(filename, "std:", "", -1)
		// Join up the new filename to the standard library source location
		base = util.StdLibFile(filename)
	}
	return filepath.Join(base, filename)
}

// PathIsDir returns if a given path is a directory or not
func PathIsDir(pth string) (bool, error) {
	fd, err := os.Open(pth)
	if err != nil {
		return false, err
	}
	defer fd.Close()
	stat, err := fd.Stat()
	if err != nil {
		return false, err
	}
	return stat.IsDir(), nil
}

// NamespaceFromNodes takes an array of nodes and returns the namespace name of them
func NamespaceFromNodes(nodes []Node) (string, error) {
	for _, n := range nodes {
		if n.Kind() == nodeNamespace {
			return n.(NamespaceNode).Name, nil
		}
	}

	return "error", fmt.Errorf("nodes have no package name")
}

// FilterNodes returns only the nodes that have the type passed in
func FilterNodes(nodes []Node, t NodeType) chan Node {
	filtered := make(chan Node)

	go func() {
		for _, n := range nodes {
			if n.Kind() == t {
				filtered <- n
			}
		}
		close(filtered)
	}()

	return filtered
}

// FilterPackagedNodes returns only the nodes that have the type passed in
func FilterPackagedNodes(nodes []*PackagedNode, t NodeType) chan *PackagedNode {
	filtered := make(chan *PackagedNode)
	go func() {
		for _, n := range nodes {
			if n.Node.Kind() == t {
				filtered <- n
			}
		}
		close(filtered)
	}()

	return filtered
}

// FilterPackagedNodesPredicate returns only the nodes that pass the test given
func FilterPackagedNodesPredicate(nodes []*PackagedNode, predicate func(n Node) bool) chan *PackagedNode {
	filtered := make(chan *PackagedNode)
	go func() {
		for _, n := range nodes {
			if predicate(n.Node) {
				filtered <- n
			}
		}
		close(filtered)
	}()

	return filtered
}

// PackagedNode wraps around a certain node and allows better codegen
// in the context of a certain package
type PackagedNode struct {
	Pkg     *Package
	Program *Program
	Node    Node
}

// Codegen will generate the node this PackagedNode wraps
func (p *PackagedNode) Codegen(prog *Program) {
	p.SetupContext()
	p.Node.Codegen(prog)
}

// SetupContext modifies the program to help with context information
func (p *PackagedNode) SetupContext() {
	p.Program.Package = p.Pkg
	p.Program.Scope.PackageName = p.Pkg.Name
}

// PackageNode takes a node, it's package and the program context
// and creates an encapsulated context for it
func PackageNode(node Node, pkg *Package, prog *Program) *PackagedNode {
	n := &PackagedNode{}
	n.Node = node
	n.Pkg = pkg
	n.Program = prog
	return n
}