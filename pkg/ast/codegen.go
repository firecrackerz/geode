package ast

import (
	"fmt"
	"os"
	"strings"

	"github.com/geode-lang/llvm/ir"
	"github.com/geode-lang/llvm/ir/constant"
	"github.com/geode-lang/llvm/ir/types"
	"github.com/geode-lang/llvm/ir/value"

	"github.com/geode-lang/geode/pkg/typesystem"
	"github.com/geode-lang/geode/pkg/util/log"
)

func parseName(combined string) (string, string) {
	var namespace, name string
	parts := strings.Split(combined, ":")
	name = parts[len(parts)-1]
	if len(parts) > 1 {
		namespace = parts[0]
	}

	return namespace, name
}

// A global number to indicate which `name index` we are on. This way,
// the mangler will never output the same name twice as this number is monotonic
var nameNumber int

func mangleName(name string) string {
	nameNumber++
	return fmt.Sprintf("%s_%d", name, nameNumber)
}

func branchIfNoTerminator(blk *ir.BasicBlock, to *ir.BasicBlock) {
	if blk.Term == nil {
		blk.NewBr(to)
	}
}

// Codegen returns some NamespaceNode's arguments
func (n NamespaceNode) Codegen(prog *Program) value.Value { return nil }

// Handle will do ast-level handling for a dependency node
func (n DependencyNode) Handle(prog *Program) value.Value {

	// abs, _ := filepath.Abs(prog.Compiler.Package.Source.Path)
	// dir := path.Dir(abs)

	fmt.Println(n.Paths)
	// for _, dp := range n.Paths {

	// 	depPath := path.Join(dir, dp)
	// 	if n.CLinkage {
	// 		prog.Compiler.Package.AddClinkage(depPath)
	// 	} else {
	// 		prog.Compiler.Package.LoadDep(depPath)
	// 	}

	// }

	return nil
}

// Codegen implements Node.Codegen for DependencyNode
func (n DependencyNode) Codegen(prog *Program) value.Value { return nil }

// Codegen implements Node.Codegen for IfNode
func (n IfNode) Codegen(prog *Program) value.Value {

	c := prog.Compiler
	predicate := n.If.Codegen(prog)
	zero := constant.NewInt(0, types.I32)
	// The name of the blocks is prefixed because
	namePrefix := fmt.Sprintf("if.%d.", n.Index)
	parentBlock := c.CurrentBlock()
	predicate = parentBlock.NewICmp(ir.IntNE, zero, createTypeCast(prog, predicate, types.I32))
	parentFunc := parentBlock.Parent

	var thenGenBlk *ir.BasicBlock
	var endBlk *ir.BasicBlock

	thenBlk := parentFunc.NewBlock(mangleName(namePrefix + "then"))

	c.genInBlock(thenBlk, func() {
		thenGenBlk = n.Then.Codegen(prog).(*ir.BasicBlock)
	})

	elseBlk := parentFunc.NewBlock(mangleName(namePrefix + "else"))
	var elseGenBlk *ir.BasicBlock

	c.genInBlock(elseBlk, func() {
		// We only want to construct the else block if there is one.
		if n.Else != nil {
			elseGenBlk = n.Else.Codegen(prog).(*ir.BasicBlock)
		}
	})

	endBlk = parentFunc.NewBlock(mangleName(namePrefix + "end"))
	c.PushBlock(endBlk)
	// We need to make sure these blocks have terminators.
	// in order to do that, we branch to the end block
	branchIfNoTerminator(thenBlk, endBlk)
	branchIfNoTerminator(thenGenBlk, endBlk)
	branchIfNoTerminator(elseBlk, endBlk)

	if elseGenBlk != nil {
		branchIfNoTerminator(elseGenBlk, endBlk)
	}

	parentBlock.NewCondBr(predicate, thenBlk, elseBlk)

	// branchIfNoTerminator(c.CurrentBlock(), endBlk)

	return endBlk
}

// Codegen implements Node.Codegen for ForNode
func (n ForNode) Codegen(prog *Program) value.Value {

	c := prog.Compiler

	// The name of the blocks is prefixed so we can determine which for loop a block is for.
	namePrefix := fmt.Sprintf("for.%X.", n.Index)
	parentBlock := c.CurrentBlock()
	prog.Scope = prog.Scope.SpawnChild()

	var predicate value.Value
	var condBlk *ir.BasicBlock
	var bodyBlk *ir.BasicBlock
	var bodyGenBlk *ir.BasicBlock
	var endBlk *ir.BasicBlock
	parentFunc := parentBlock.Parent

	condBlk = parentFunc.NewBlock(namePrefix + "cond")

	n.Init.Codegen(prog)

	parentBlock.NewBr(condBlk)

	c.genInBlock(condBlk, func() {
		predicate = n.Cond.Codegen(prog)
		one := constant.NewInt(1, types.I1)
		predicate = condBlk.NewICmp(ir.IntEQ, one, createTypeCast(prog, predicate, types.I1))
	})
	bodyBlk = parentFunc.NewBlock(namePrefix + "body")
	c.genInBlock(bodyBlk, func() {
		bodyGenBlk = n.Body.Codegen(prog).(*ir.BasicBlock)

		c.genInBlock(bodyGenBlk, func() {
			n.Step.Codegen(prog)
		})
		branchIfNoTerminator(bodyBlk, condBlk)
		branchIfNoTerminator(bodyGenBlk, condBlk)
	})
	endBlk = parentFunc.NewBlock(namePrefix + "end")
	c.PushBlock(endBlk)
	condBlk.NewCondBr(predicate, bodyBlk, endBlk)

	prog.Scope = prog.Scope.Parent
	return endBlk
}

// Codegen implements Node.Codegen for CharNode
func (n CharNode) Codegen(prog *Program) value.Value {
	return constant.NewInt(int64(n.Value), types.I8)
}

func (n CharNode) GenAccess(prog *Program) value.Value {
	return n.Codegen(prog)
}

// Codegen implements Node.Codegen for UnaryNode
func (n UnaryNode) Codegen(prog *Program) value.Value {

	c := prog.Compiler

	operandValue := n.Operand.Codegen(prog)
	if operandValue == nil {
		n.Operand.SyntaxError()
		log.Fatal("nil operand")
	}

	if n.Operator == "-" {

		if types.IsFloat(operandValue.Type()) {
			return c.CurrentBlock().NewFSub(constant.NewFloat(0, types.Double), operandValue)
		} else if types.IsInt(operandValue.Type()) {
			return c.CurrentBlock().NewSub(constant.NewInt(0, types.I64), operandValue)
		}
		n.SyntaxError()
		log.Fatal("Unable to make a non integer/float into a negative\n")

	}

	// handle reference operation
	if n.Operator == "&" {
		return operandValue
	}
	// handle dereference operation
	if n.Operator == "*" {
		if types.IsPointer(operandValue.Type()) {
			return c.CurrentBlock().NewLoad(operandValue)
		}
		n.SyntaxError()
		log.Fatal("attempt to dereference a non-pointer variable\n")
	}

	return operandValue
}

// GenAccess implements Accessable.GenAccess
func (n UnaryNode) GenAccess(prog *Program) value.Value {
	return n.Codegen(prog)
}

// Codegen implements Node.Codegen for WhileNode
func (n WhileNode) Codegen(prog *Program) value.Value {

	c := prog.Compiler

	// The name of the blocks is prefixed because
	namePrefix := fmt.Sprintf("while_%d_", n.Index)
	parentBlock := c.CurrentBlock()

	parentFunc := parentBlock.Parent
	startblock := parentFunc.NewBlock(mangleName(namePrefix + "start"))
	c.PushBlock(startblock)
	predicate := n.If.Codegen(prog)
	one := constant.NewInt(1, types.I1)
	c.PopBlock()
	branchIfNoTerminator(parentBlock, startblock)
	predicate = startblock.NewICmp(ir.IntEQ, one, createTypeCast(prog, predicate, types.I1))

	var endBlk *ir.BasicBlock

	bodyBlk := parentFunc.NewBlock(mangleName(namePrefix + "body"))
	c.PushBlock(bodyBlk)
	bodyGenBlk := n.Body.Codegen(prog).(*ir.BasicBlock)

	// If there is no terminator for the block, IE: no return
	// branch to the merge block

	endBlk = parentFunc.NewBlock(mangleName(namePrefix + "merge"))
	c.PushBlock(endBlk)

	branchIfNoTerminator(bodyBlk, startblock)
	branchIfNoTerminator(bodyGenBlk, startblock)

	startblock.NewCondBr(predicate, bodyBlk, endBlk)

	// branchIfNoTerminator(c.CurrentBlock(), endBlk)

	return endBlk
}

func typeSize(t types.Type) int {
	if types.IsInt(t) {
		return t.(*types.IntType).Size
	}
	if types.IsFloat(t) {
		return int(t.(*types.FloatType).Kind)
	}

	return -1
}

func binaryCast(prog *Program, left, right value.Value) (value.Value, value.Value, types.Type) {
	// Right and Left types
	lt := left.Type()
	rt := right.Type()

	var casted types.Type

	// Get the cast precidence of both sides
	leftPrec := typesystem.CastPrecidence(lt)
	rightPrec := typesystem.CastPrecidence(rt)

	if leftPrec > rightPrec {
		casted = lt
		right = createTypeCast(prog, right, lt)
	} else {
		casted = rt
		left = createTypeCast(prog, left, rt)
	}
	return left, right, casted
}

// createTypeCast is where most, if not all, type casting happens in the language.
func createTypeCast(prog *Program, in value.Value, to types.Type) value.Value {
	c := prog.Compiler
	inType := in.Type()
	fromInt := types.IsInt(inType)
	fromFloat := types.IsFloat(inType)

	toInt := types.IsInt(to)
	toFloat := types.IsFloat(to)

	inSize := typeSize(inType)
	outSize := typeSize(to)

	if types.Equal(to, types.Void) {
		return nil
	}

	if types.IsPointer(inType) && types.IsPointer(to) {
		return c.CurrentBlock().NewBitCast(in, to)
	}

	if fromFloat && toInt {
		return c.CurrentBlock().NewFPToSI(in, to)
	}

	if fromInt && toFloat {
		return c.CurrentBlock().NewSIToFP(in, to)
	}

	if fromInt && toInt {
		if inSize < outSize {
			return c.CurrentBlock().NewSExt(in, to)
		}
		if inSize == outSize {
			return in
		}
		return c.CurrentBlock().NewTrunc(in, to)
	}

	if fromFloat && toFloat {
		if inSize < outSize {
			return c.CurrentBlock().NewFPExt(in, to)
		}
		if inSize == outSize {
			return in
		}
		return c.CurrentBlock().NewFPTrunc(in, to)
	}

	// If the cast would not change the type, just return the in value
	if types.Equal(inType, to) {
		return in
	}

	log.Fatal("Failed to typecast type %s to %s\n", inType.String(), to)
	return nil
}

func createCmp(blk *ir.BasicBlock, i ir.IntPred, f ir.FloatPred, t types.Type, left, right value.Value) value.Value {
	if types.IsInt(t) {
		return blk.NewICmp(i, left, right)
	}
	if types.IsFloat(t) {
		return blk.NewFCmp(f, left, right)
	}
	log.Fatal("Creation of rem instruction failed. `%s % %s`\n", left.Type(), right.Type())
	return nil
}

// CreateBinaryOp produces a geode binary op (just a wrapper around geode-lang/llvm's binary instructions)
func CreateBinaryOp(intstr, fltstr string, blk *ir.BasicBlock, t types.Type, left, right value.Value) value.Value {
	var inst *GeodeBinaryInstr
	if types.IsInt(t) {
		inst = NewGeodeBinaryInstr(intstr, left, right)
	} else {
		inst = NewGeodeBinaryInstr(fltstr, left, right)
	}
	blk.AppendInst(inst)
	return inst
}

// Codegen implements Node.Codegen for BinaryNode
func (n BinaryNode) Codegen(prog *Program) value.Value {

	// Generate the left and right nodes
	l := n.Left.Codegen(prog)
	r := n.Right.Codegen(prog)

	// Attempt to cast them with casting precidence
	// This means the operation `int + float` will cast the int to a float.
	l, r, t := binaryCast(prog, l, r)

	if l == nil || r == nil {
		n.SyntaxError()
		log.Fatal("An operand to a binary operation `%s` was nil and failed to generate\n", n.OP)
	}

	blk := prog.Compiler.CurrentBlock()

	switch n.OP {
	case "+":
		return CreateBinaryOp("add", "fadd", blk, t, l, r)
	case "-":
		return CreateBinaryOp("sub", "fsub", blk, t, l, r)
	case "*":
		return CreateBinaryOp("mul", "fmul", blk, t, l, r)
	case "/":
		return CreateBinaryOp("sdiv", "fdiv", blk, t, l, r)
	case "%":
		return CreateBinaryOp("srem", "frem", blk, t, l, r)
	case ">>":
		return CreateBinaryOp("lshr", "lshr", blk, t, l, r)
	case "<<":
		return CreateBinaryOp("shl", "shl", blk, t, l, r)
	case "||":
		return CreateBinaryOp("or", "or", blk, t, l, r)
	case "&&":
		return CreateBinaryOp("and", "and", blk, t, l, r)
	case "^":
		return CreateBinaryOp("xor", "xor", blk, t, l, r)

	case "=":
		return createCmp(blk, ir.IntEQ, ir.FloatOEQ, t, l, r)
	case "!=", "≠":
		return createCmp(blk, ir.IntNE, ir.FloatONE, t, l, r)
	case ">":
		return createCmp(blk, ir.IntSGT, ir.FloatOGT, t, l, r)

	case ">=":
		return createCmp(blk, ir.IntSGE, ir.FloatOGE, t, l, r)
	case "<":
		return createCmp(blk, ir.IntSLT, ir.FloatOLT, t, l, r)
	case "<=":
		return createCmp(blk, ir.IntSLE, ir.FloatOLE, t, l, r)
	default:
		return codegenError("invalid binary operator")
	}
}

// Codegen implements Node.Codegen for FunctionCallNode
func (n FunctionCallNode) Codegen(prog *Program) value.Value {

	scope := prog.Scope
	c := prog.Compiler

	// scopeItem, found := c.Scope.Find(n.Name)

	args := []value.Value{}
	argTypes := []types.Type{}
	argStrings := []string{}
	for _, arg := range n.Args {

		// fmt.Println(i)

		if ac, isAccessable := arg.(Accessable); isAccessable {
			// argStrings = append(argStrings, ac.(fmt.Stringer).String())
			val := ac.GenAccess(prog)

			args = append(args, val)
			argTypes = append(argTypes, val.Type())
			if args[len(args)-1] == nil {
				return codegenError(fmt.Sprintf("Argument to function %q failed to generate code", n.Name))
			}
		} else {
			arg.SyntaxError()
			log.Fatal("Argument to function call to '%s' is not accessable (has no readable value). Node type %s\n", n.Name, arg.Kind())
		}

	}

	// First we need to check if the function call is actually a call to a class's constructor.
	// Because in geode, calling a class name results in the constructor being called for said class.
	class := c.Scope.FindType(n.Name.String())
	if class != nil {
		return GenerateClassConstruction(n.Name.String(), class.Type, scope, c, args)
	}

	ns, nm := parseName(n.Name.String())

	if ns == "" {
		ns = c.Scope.PackageName
	} else if !prog.Package.HasAccessToPackage(ns) {
		n.SyntaxError()
		log.Fatal("Package %s doesn't load package %s but attempts to call %s:%s.\n", c.Scope.PackageName, ns, ns, nm)
	}

	completeName := fmt.Sprintf("%s:%s", ns, nm)

	name := MangleFunctionName(completeName, argTypes, n.Generics)

	functionOptions, _ := c.Scope.FindFunctions(name)
	funcCount := len(functionOptions)

	if funcCount > 1 {
		n.SyntaxError()
		log.Fatal("Too many options for function call '%s'\n", name)
	} else if funcCount == 0 {
		unmangled := UnmangleFunctionName(name)
		// _, bareName := parseName(unmangled)

		n.SyntaxError()
		log.Fatal("Unable to find function '%s' in scope of module '%s'. Mangled name: %q\n", unmangled, c.Scope.PackageName, name)
	}

	fnScopeItem := functionOptions[0]
	if !c.FunctionDefined(fnScopeItem.function) {
		c.Module.AppendFunction(fnScopeItem.function)
	}

	callee := fnScopeItem.Value().(*ir.Function)
	if callee == nil {
		return codegenError(fmt.Sprintf("Unknown function %q referenced", name))
	}

	// Attempt to typecast all the args into the correct type
	// This is skipped with variadic functions
	if !callee.Sig.Variadic {
		for i := range args {
			args[i] = createTypeCast(prog, args[i], callee.Sig.Params[i].Type())
		}
	}

	c.CurrentBlock().AppendInst(NewLLVMComment("%s(%s);", completeName, strings.Join(argStrings, ", ")))
	return c.CurrentBlock().NewCall(callee, args...)
}

// Codegen implements Node.Codegen for ReturnNode
func (n ReturnNode) Codegen(prog *Program) value.Value {
	c := prog.Compiler

	var retVal value.Value

	if c.FN.Sig.Ret != types.Void {
		if n.Value != nil {
			retVal = n.Value.Codegen(prog)
			// retVal = createTypeCast(c, retVal, c.FN.Sig.Ret)
		} else {
			retVal = nil
		}
	}

	c.CurrentBlock().NewRet(retVal)

	return retVal
}

func newCharArray(s string) *constant.Array {
	var bs []constant.Constant
	for i := 0; i < len(s); i++ {
		b := constant.NewInt(int64(s[i]), types.I8)
		bs = append(bs, b)
	}
	bs = append(bs, constant.NewInt(0, types.I8))
	c := constant.NewArray(bs...)
	c.CharArray = true
	return c
}

// CreateEntryBlockAlloca - Create an alloca instruction in the entry block of
// the function.  This is used for mutable variables etc.
func createBlockAlloca(f *ir.Function, elemType types.Type, name string) *ir.InstAlloca {
	// Create a new allocation in the root of the function
	alloca := f.Blocks[0].NewAlloca(elemType)
	// Set the name of the allocation (the variable name)
	alloca.SetName(name)
	return alloca
}

// Allow functions to return an error isntead of having to manage closing the program each time.
func codegenError(str string, args ...interface{}) value.Value {
	fmt.Fprintf(os.Stderr, "Error: %s\n", fmt.Sprintf(str, args...))
	return nil
}
