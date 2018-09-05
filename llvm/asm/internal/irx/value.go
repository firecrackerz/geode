package irx

import (
	"fmt"

	"github.com/geode-lang/geode/llvm/asm/internal/ast"
	"github.com/geode-lang/geode/llvm/ir"
	"github.com/geode-lang/geode/llvm/ir/value"
)

// irValue returns the corresponding LLVM IR value of the given value.
func (m *Module) irValue(old ast.Value) value.Value {
	switch old := old.(type) {
	// Constant.
	case ast.Constant:
		return m.irConstant(old)
	// Named values.
	case ast.NamedValue:
		switch old := old.(type) {
		// Global identifiers.
		case *ast.Global, *ast.GlobalDummy, *ast.Function:
			return m.getGlobal(old.GetName())
		// Local identifiers.
		case *ast.Param, *ast.BasicBlock, *ast.LocalDummy, ast.Instruction:
			return m.getLocal(old.GetName())
		default:
			panic(fmt.Errorf("support for named value %T not yet implemented", old))
		}
	// Inline assmebly.
	case *ast.InlineAsm:
		return &ir.InlineAsm{
			Asm:         old.Asm,
			Constraints: old.Constraints,
			Typ:         m.irType(old.Type),
		}
	// Metadata node.
	case ast.MetadataNode:
		return m.irMetadataNode(old)
	default:
		panic(fmt.Errorf("support for value %T not yet implemented", old))
	}
}
