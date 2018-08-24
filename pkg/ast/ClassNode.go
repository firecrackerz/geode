package ast

import (
	"fmt"

	"github.com/geode-lang/geode/pkg/util/color"
	"github.com/geode-lang/geode/pkg/util/log"
	"github.com/geode-lang/llvm/ir"
	"github.com/geode-lang/llvm/ir/constant"
	"github.com/geode-lang/llvm/ir/types"
	"github.com/geode-lang/llvm/ir/value"
)

// ClassNode -
type ClassNode struct {
	NodeType
	TokenReference

	Name      string
	Methods   []FunctionNode
	Variables []VariableDefnNode
}

// NameString implements Node.NameString
func (n ClassNode) NameString() string { return "ClassNode" }

// InferType implements Node.InferType
func (n ClassNode) InferType(scope *Scope) string { return "void" }

// VerifyCorrectness checks if a class will cause any problems when we pass it off to clang
// some problems might include the following:
// -  class Foo {
//        Foo a; # this will take infinite stack space when allocated.
//    }
// -  class Foo {
//        Bar a;
//    }
//    class Bar {
//        Foo b;
//    }
func (n ClassNode) VerifyCorrectness(scope *Scope, c *Compiler) bool {
	base := scope.FindType(n.Name).Type.(*types.StructType)

	for _, f := range n.Variables {
		fieldName := f.Name.String()
		t := f.Type.Name
		ty := scope.FindType(t).Type
		ty = f.Type.BuildPointerType(ty)

		// Pointer types should be correct
		if types.IsPointer(ty) {
			continue
		}

		if types.IsStruct(ty) {
			// If the type is a direct reference back to the base class, it is invalid. It must be a pointer type
			if types.Equal(base, ty) {
				fmt.Printf("Class '%s' has a circular reference in it's fields. Field '%s' should be a pointer to a '%s' instead.\n", n.Name, f.Name, n.Name)
				return false
			}

			// Now we need to check if the struct has a non-pointer reference back to this class.
			// that has the same effect.
			structT := ty.(*types.StructType)

			if contains, _, _ := structContainsTypeAnywhere(structT, base, structT); contains {
				fmt.Printf("* Problem: class %s has a field %s of type %s which eventually back references %s (would consume 'infinite' stack memory)\n", color.Green(n.Name), color.Blue(fieldName), color.Red(t), color.Green(n.Name))
				fmt.Printf("  Solution: Either change %s to a pointer or remove the back-reference from %s\n\n", color.Blue(fieldName), color.Red(t))
				return false
			}
		}
	}
	return true
}

func structContainsTypeAnywhere(s *types.StructType, t types.Type, path ...*types.StructType) (bool, int, []*types.StructType) {
	for i, field := range s.Fields {
		if types.Equal(field, t) {
			return true, i, path
		}
		if types.IsStruct(field) {
			structType := field.(*types.StructType)
			if contains, index, p := structContainsTypeAnywhere(structType, t, append(path, structType)...); contains {
				return true, index, p
			}
		}
	}
	return false, -1, nil
}

func (n ClassNode) String() string {
	return fmt.Sprintf("class %s {}", n.Name)
}

// Declare a class type
func (n ClassNode) Declare(prog *Program) value.Value {
	structDefn := types.NewStruct()

	prog.Scope = prog.Scope.SpawnChild()

	name := fmt.Sprintf("class.%s.%s", prog.Scope.PackageName, n.Name)
	structDefn.SetName(name)

	prog.Module.NewType(n.Name, structDefn)
	prog.Scope.GetRoot().RegisterType(n.Name, structDefn, -1)
	// structDefn.Opaque = true

	prog.Scope = prog.Scope.Parent

	return nil
}

// Codegen implements Node.Codegen for ClassNode
func (n ClassNode) Codegen(prog *Program) value.Value {

	structDefn := prog.Scope.FindType(n.Name).Type.(*types.StructType)

	fieldnames := make([]string, 0, len(n.Variables))
	fields := make([]types.Type, 0, len(n.Variables))

	names := map[string]bool{}

	for _, f := range n.Variables {
		t := f.Type.Name
		name := f.Name.String()
		if _, found := names[name]; found {
			log.Fatal("Class '%s' has two fields/methods named '%s'\n", n.Name, f.Name)
		}
		names[name] = true
		fmt.Println(t)
		ty := prog.Scope.FindType(t).Type
		ty = f.Type.BuildPointerType(ty)
		fields = append(fields, ty)
		fieldnames = append(fieldnames, name)
	}

	thisArg := VariableDefnNode{}
	thisArg.Name = NewNamedReference("this")
	thisArg.Type = GeodeTypeRef{}
	thisArg.Type.Name = n.Name
	thisArg.Type.PointerLevel = 1

	structDefn.Fields = fields
	structDefn.Names = fieldnames

	// methodBaseArgs := []VariableDefnNode{thisArg}
	for _, fn := range n.Methods {
		fn.Name.Value = fmt.Sprintf("%s:%s.%s", prog.Package.Name, n.Name, fn.Name)

		if _, found := names[fn.Name.String()]; found {
			log.Fatal("Class '%s' has two fields/methods named '%s'\n", n.Name, fn.Name)
		}
		names[fn.Name.String()] = true
		prog.RegisterFunction(fn.Name.Value, fn)
		// prog.CompileFunction(fn.Name.Value)

		// m.Args = append(methodBaseArgs, m.Args...)
		// m.Declare(prog)
		// m.Codegen(prog)
	}

	return nil
}

// GenerateClassConstruction creates a function call to a class's constructor if it exists.
func GenerateClassConstruction(name string, typ types.Type, s *Scope, c *Compiler, args []value.Value) value.Value {
	alloc := c.CurrentBlock().NewAlloca(typ)

	load := c.CurrentBlock().NewLoad(alloc)
	return load
}

// NewClassInstance takes the class to generate as well as the fields
// mapped to their value
func NewClassInstance(prog *Program, stct *types.StructType, fields map[string]value.Value) value.Value {

	alloc := prog.Compiler.CurrentBlock().NewAlloca(stct)

	for field, value := range fields {
		GenStructFieldAssignment(prog, alloc, field, value)
	}

	load := prog.Compiler.CurrentBlock().NewLoad(alloc)

	return load
}

// GetStructFieldAlloc returns the allocation offset of some struct instance
func GetStructFieldAlloc(prog *Program, alloc *ir.InstAlloca, field string) value.Value {
	var base value.Value

	base = alloc
	baseType := GetBaseType(base)

	ptr := alloc.Type().(*types.PointerType)
	elemType := ptr.Elem

	// If the type that the alloca points to is a pointer, we need to load from the pointer
	if types.IsPointer(elemType) {
		base = prog.Compiler.CurrentBlock().NewLoad(base)
	}

	structType := baseType.(*types.StructType)

	index := structType.FieldIndex(field)

	zero := constant.NewInt(0, types.I32)
	fieldOffset := constant.NewInt(int64(index), types.I32)
	gen := prog.Compiler.CurrentBlock().NewGetElementPtr(base, zero, fieldOffset)
	return gen
}

// GenStructFieldAssignment takes some allocation and assigns the value to a field given some name
func GenStructFieldAssignment(prog *Program, alloc *ir.InstAlloca, field string, val value.Value) {
	gen := GetStructFieldAlloc(prog, alloc, field)
	prog.Compiler.CurrentBlock().NewStore(gen, val)
}

// GetBaseType returns the base type of some alloca
func GetBaseType(v value.Value) types.Type {
	base := v.(*ir.InstAlloca)
	baseType := base.Elem
	for types.IsPointer(baseType) {
		baseType = baseType.(*types.PointerType).Elem
	}
	return baseType
}
