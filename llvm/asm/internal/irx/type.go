package irx

import (
	"fmt"

	"github.com/geode-lang/geode/llvm/asm/internal/ast"
	"github.com/geode-lang/geode/llvm/ir/types"
)

// irType returns the corresponding LLVM IR type of the given type.
func (m *Module) irType(old ast.Type) types.Type {
	switch old := old.(type) {
	case *ast.VoidType:
		return types.Void
	case *ast.FuncType:
		params := make([]*types.Param, len(old.Params))
		for i, oldParam := range old.Params {
			params[i] = types.NewParam(oldParam.Name, m.irType(oldParam.Type))
		}
		typ := types.NewFunc(m.irType(old.Ret), params...)
		typ.Variadic = old.Variadic
		return typ
	case *ast.IntType:
		return types.NewInt(old.Size)
	case *ast.FloatType:
		switch old.Kind {
		case ast.FloatKindIEEE_16:
			return types.Half
		case ast.FloatKindIEEE_32:
			return types.Float
		case ast.FloatKindIEEE_64:
			return types.Double
		case ast.FloatKindIEEE_128:
			return types.FP128
		case ast.FloatKindDoubleExtended_80:
			return types.X86_FP80
		case ast.FloatKindDoubleDouble_128:
			return types.PPC_FP128
		default:
			panic(fmt.Errorf("support for %v not yet implemented", old.Kind))
		}
	case *ast.PointerType:
		typ := types.NewPointer(m.irType(old.Elem))
		typ.AddrSpace = old.AddrSpace
		return typ
	case *ast.VectorType:
		return types.NewVector(m.irType(old.Elem), old.Len)
	case *ast.LabelType:
		return types.Label
	case *ast.MetadataType:
		return types.Metadata
	case *ast.ArrayType:
		return types.NewArray(m.irType(old.Elem), old.Len)
	case *ast.StructType:
		fields := make([]types.Type, len(old.Fields))
		for i, oldField := range old.Fields {
			fields[i] = m.irType(oldField)
		}
		typ := types.NewStruct(fields...)
		typ.Opaque = old.Opaque
		return typ
	case *ast.NamedType:
		return m.getType(old.Name)
	case *ast.NamedTypeDummy:
		return m.getType(old.Name)
	case *ast.TypeDummy:
		panic("invalid type *ast.TypeDummy; dummy types should have been translated during parsing by astx")
	default:
		panic(fmt.Errorf("support for %T not yet implemented", old))
	}
}

// aggregateElemType returns the element type of the given aggregate type, based
// on the specified indices.
func aggregateElemType(t types.Type, indices []int64) types.Type {
	if len(indices) == 0 {
		return t
	}
	index := indices[0]
	switch t := t.(type) {
	case *types.ArrayType:
		if index >= t.Len {
			panic(fmt.Errorf("invalid index (%d); exceeds array length (%d)", index, t.Len))
		}
		return aggregateElemType(t.Elem, indices[1:])
	case *types.StructType:
		if index >= int64(len(t.Fields)) {
			panic(fmt.Errorf("invalid index (%d); exceeds struct field count (%d)", index, len(t.Fields)))
		}
		return aggregateElemType(t.Fields[index], indices[1:])
	default:
		panic(fmt.Errorf("invalid aggregate value type; expected *types.ArrayType or *types.StructType, got %T", t))
	}
}
