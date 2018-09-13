package irx

import (
	"fmt"

	"github.com/geode-lang/geode/llvm/asm/internal/ast"
	"github.com/geode-lang/geode/llvm/ir"
	"github.com/geode-lang/geode/llvm/ir/metadata"
	"github.com/geode-lang/geode/llvm/ir/types"
)

// ### [ Helper functions ] ####################################################

// irIntPred returns the corresponding LLVM IR integer predicate of the given
// integer predicate.
func irIntPred(cond ast.IntPred) ir.IntPred {
	switch cond {
	case ast.IntEQ:
		return ir.IntEQ
	case ast.IntNE:
		return ir.IntNE
	case ast.IntUGT:
		return ir.IntUGT
	case ast.IntUGE:
		return ir.IntUGE
	case ast.IntULT:
		return ir.IntULT
	case ast.IntULE:
		return ir.IntULE
	case ast.IntSGT:
		return ir.IntSGT
	case ast.IntSGE:
		return ir.IntSGE
	case ast.IntSLT:
		return ir.IntSLT
	case ast.IntSLE:
		return ir.IntSLE
	}
	panic(fmt.Errorf("support for integer predicate %v not yet implemented", cond))
}

// irFloatPred returns the corresponding LLVM IR floating-point predicate of the
// given floating-point predicate.
func irFloatPred(cond ast.FloatPred) ir.FloatPred {
	switch cond {
	case ast.FloatFalse:
		return ir.FloatFalse
	case ast.FloatOEQ:
		return ir.FloatOEQ
	case ast.FloatOGT:
		return ir.FloatOGT
	case ast.FloatOGE:
		return ir.FloatOGE
	case ast.FloatOLT:
		return ir.FloatOLT
	case ast.FloatOLE:
		return ir.FloatOLE
	case ast.FloatONE:
		return ir.FloatONE
	case ast.FloatORD:
		return ir.FloatORD
	case ast.FloatUEQ:
		return ir.FloatUEQ
	case ast.FloatUGT:
		return ir.FloatUGT
	case ast.FloatUGE:
		return ir.FloatUGE
	case ast.FloatULT:
		return ir.FloatULT
	case ast.FloatULE:
		return ir.FloatULE
	case ast.FloatUNE:
		return ir.FloatUNE
	case ast.FloatUNO:
		return ir.FloatUNO
	case ast.FloatTrue:
		return ir.FloatTrue
	}
	panic(fmt.Errorf("support for floating-point predicate %v not yet implemented", cond))
}

// irMetadata returns the corresponding LLVM IR metadata of the given list of
// attached metadata.
func (m *Module) irMetadata(oldMDs []*ast.AttachedMD) map[string]*metadata.Metadata {
	mds := make(map[string]*metadata.Metadata)
	for _, oldMD := range oldMDs {
		key := oldMD.Name
		node := m.metadataNode(oldMD.Metadata)
		if prev, ok := mds[key]; ok {
			panic(fmt.Errorf("attached metadata for metadata name %q already present; previous `%v`, new `%v`", key, prev, m.Metadata))
		}
		md, ok := node.(*metadata.Metadata)
		if !ok {
			panic(fmt.Errorf("invalid metadata type; expected *metadata.Metadata, got %T", node))
		}
		mds[key] = md
	}
	return mds
}

// newEmptyNamedType returns an empty type definition for the given named type.
func newEmptyNamedType(old ast.Type) types.Type {
	switch old := old.(type) {
	case *ast.VoidType:
		return &types.VoidType{}
	case *ast.FuncType:
		return &types.FuncType{}
	case *ast.IntType:
		return &types.IntType{}
	case *ast.FloatType:
		return &types.FloatType{}
	case *ast.PointerType:
		return &types.PointerType{}
	case *ast.VectorType:
		return &types.VectorType{}
	case *ast.LabelType:
		return &types.LabelType{}
	case *ast.MetadataType:
		return &types.MetadataType{}
	case *ast.ArrayType:
		return &types.ArrayType{}
	case *ast.StructType:
		return &types.StructType{}
	case *ast.NamedType:
		return newEmptyNamedType(old.Def)
	default:
		panic(fmt.Errorf("support for type %T not yet implemented", old))
	}
}
