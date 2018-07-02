package ast

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strings"
	"time"

	"github.com/llir/llvm/ir"
	"github.com/nickwanninger/geode/pkg/lexer"
	"github.com/nickwanninger/geode/pkg/util/log"
)

// RuntimePackage is the global runtime package
var RuntimePackage *Package
var dependencyMap map[string]*Package

func init() {
	RuntimePackage = GetRuntime()
	dependencyMap = make(map[string]*Package)
}

// Package is a wrapper around a module. It is able
// to compile and emit code, as well as lex and parse it.
type Package struct {
	fmt.Stringer

	Name               string
	Lexer              *lexer.LexState
	Source             *lexer.Sourcefile
	Nodes              []Node
	Dependencies       []*Package
	Scope              *Scope
	Compiler           *Compiler
	IsRuntime          bool
	objectFilesEmitted []string
	Compiled           bool
	CLinkages          []string
}

// NewPackage returns a pointer to a new package
func NewPackage(name string, source *lexer.Sourcefile) *Package {
	p := &Package{}

	p.Name = name
	p.Source = source
	p.Nodes = make([]Node, 0)
	p.Scope = NewScope()
	p.Lexer = lexer.NewLexer()

	return p
}

// String will get the LLVM IR from the package's compiler
func (p *Package) String() string {
	ir := ""
	// We need to build up the IR that will be emitted
	// so we can track this information later on.
	ir += fmt.Sprintf("; ModuleID = %q\n", p.Name)
	ir += fmt.Sprintf("; SourceHash = %x\n", p.Hash())
	ir += fmt.Sprintf("; UnixDate = %d\n", time.Now().Unix())
	ir += fmt.Sprintf("source_filename = %q\n", p.Source.Path)

	ir += "\n"
	// Append the module information
	ir += fmt.Sprintf("%s\n", p.Compiler.Module.String())

	return ir
}

// Emit will emit the package as IR to a file for further compiling
func (p *Package) Emit() string {
	name := strings.Replace(p.Name, ".g", "", -1)
	filename := fmt.Sprintf("%s.%x.ll", name, p.Hash())
	ir := p.String()

	writeErr := ioutil.WriteFile(filename, []byte(ir), 0666)
	if writeErr != nil {
		panic(writeErr)
	}

	p.objectFilesEmitted = append(p.objectFilesEmitted, filename)
	return filename
}

// Hash returns the truncated sha1 of the soruce file
func (p *Package) Hash() []byte {
	return p.Source.Hash()
}

// AddDepPackage appends a dependency from a pacakge
func (p *Package) AddDepPackage(pkg *Package) {
	// Here I check for circular dependencies, which are not allowed
	sourceHash := p.Source.HashName()
	for _, dep := range pkg.Dependencies {
		if dep.Source.HashName() == sourceHash {
			log.Fatal("Circular dependency detected: %s <-> %s\n", pkg.Name, p.Name)
		}
	}
	p.Dependencies = append(p.Dependencies, pkg)
}

// AddClinkage - takes an absolute path to a c file, and adds it to the link list
func (p *Package) AddClinkage(libPath string) {
	p.CLinkages = append(p.CLinkages, libPath)
}

// LoadDep appends a dependency from a path
func (p *Package) LoadDep(depPath string) {
	filename := path.Base(depPath)

	if strings.HasPrefix(filename, "std::") {
		filename = strings.Replace(filename, "std::", "", -1)
		gopath := os.Getenv("GOPATH")
		// Join up the new filename to the standard library source location
		depPath = path.Join(gopath, "/src/github.com/nickwanninger/geode/lib/", filename)
	}

	depSource, err := lexer.NewSourcefile(filename)

	if err != nil {
		log.Fatal("Error creating dependency source structure\n")
	}
	depSource.ResolveFile(depPath)

	pkgName := fmt.Sprintf("%s_%x", filename, depSource.Hash())

	if pkg, ok := dependencyMap[depSource.HashName()]; ok {
		p.AddDepPackage(pkg)
		return
	}

	depPkg := NewPackage(pkgName, depSource)
	for _ = range depPkg.Parse() {
	}
	dependencyMap[depPkg.Source.HashName()] = depPkg
	p.AddDepPackage(depPkg)
}

// InjectExternalFunction injects the function without the body, just the sig
func (p *Package) InjectExternalFunction(fn *ir.Function) {
	ex := ir.NewFunction(fn.Name, fn.Sig.Ret, fn.Params()...)
	ex.Sig.Variadic = fn.Sig.Variadic
	scopeItem := NewFunctionScopeItem(fn.Name, ex, PublicVisibility)
	p.Scope.Add(scopeItem)

}

// Inject another Package's defintions into this Package
// This is how external dependencies work
func (p *Package) Inject(c *Package) {
	p.Dependencies = append(p.Dependencies, c)
	// Copy over all Scope Variables
	for _, v := range c.Scope.Vals {
		if v.Visibility() == PublicVisibility {

			if v.Type() == ScopeItemFunctionType {
				// fmt.Println(p.Name, v.Name())
				p.InjectExternalFunction(v.Value().(*ir.Function))
			} else {
				p.Scope.Add(v)
			}

		}
	}
}

// Parse returns a channel of new packages that will be compiled.
func (p *Package) Parse() chan *Package {

	chn := make(chan *Package)
	go func() {
		// Pull the source bytes out
		srcBytes := p.Source.Bytes()
		// go and lex the bytes
		go p.Lexer.Lex(srcBytes) // run the lexer
		// Parse the bytes into a channel of nodes
		nodes := Parse(p.Lexer.Tokens)
		// And append all those nodes to the package's nodes.
		for node := range nodes {
			p.Nodes = append(p.Nodes, node)
		}

		chn <- p
		close(chn)
	}()
	return chn
}

// GetRuntime builds a runtime
func GetRuntime() *Package {
	rts, err := lexer.NewSourcefile("runtime")
	if err != nil {
		log.Fatal("Error creating runtime source structure\n")
	}
	gopath := os.Getenv("GOPATH")
	rts.LoadFile(gopath + "/src/github.com/nickwanninger/geode/lib/runtime.g")
	rt := NewPackage("runtime", rts)
	rt.IsRuntime = true
	for _ = range rt.Parse() {
	}

	return rt
}

// Compile returns a codegen-ed compiler instance
func (p *Package) Compile() chan *Package {
	packages := make(chan *Package)

	go func() {
		p.Compiler = NewCompiler(p.Name, p)

		if !p.IsRuntime {
			p.AddDepPackage(RuntimePackage)
		}
		// Go through all nodes and handle the ones that are dependencies
		for _, node := range p.Nodes {
			if node.Kind() == nodeDependency {
				node.(dependencyNode).Handle(p.Compiler)
			}
		}

		for _, dep := range p.Dependencies {
			if !dep.Compiled {
				dep.Compiled = true
				for pkg := range dep.Compile() {
					packages <- pkg
				}
			}
			p.Inject(dep)
		}
		p.Compiled = true

		// First we *Need* to go through and declare all the functions
		for _, node := range p.Nodes {
			if node.Kind() == nodeFunction {
				node.(functionNode).Declare(p.Scope.SpawnChild(), p.Compiler)
			}
			// node.Codegen(p.Compiler.Scope.SpawnChild(), p.Compiler)
		}

		for _, node := range p.Nodes {
			// node.Codegen(p.Compiler.Scope, p.Compiler)
			node.Codegen(p.Compiler.Scope.SpawnChild(), p.Compiler)
		}

		packages <- p
		close(packages)
	}()

	return packages
}