// === [ Other instructions ] ==================================================
//
// References:
//    http://llvm.org/docs/LangRef.html#other-operations

package ast

// --- [ icmp ] ----------------------------------------------------------------

// InstICmp represents an icmp instruction.
//
// References:
//    http://llvm.org/docs/LangRef.html#icmp-instruction
type InstICmp struct {
	// Name of the local variable associated with the instruction.
	Name string
	// Integer predicate.
	Pred IntPred
	// Operands.
	X, Y Value
	// Metadata attached to the instruction.
	Metadata []*AttachedMD
}

// GetName returns the name of the value.
func (inst *InstICmp) GetName() string {
	return inst.Name
}

// SetName sets the name of the value.
func (inst *InstICmp) SetName(name string) {
	inst.Name = name
}

// --- [ fcmp ] ----------------------------------------------------------------

// InstFCmp represents an fcmp instruction.
//
// References:
//    http://llvm.org/docs/LangRef.html#fcmp-instruction
type InstFCmp struct {
	// Name of the local variable associated with the instruction.
	Name string
	// Floating-point predicate.
	Pred FloatPred
	// Operands.
	X, Y Value
	// Metadata attached to the instruction.
	Metadata []*AttachedMD
}

// GetName returns the name of the value.
func (inst *InstFCmp) GetName() string {
	return inst.Name
}

// SetName sets the name of the value.
func (inst *InstFCmp) SetName(name string) {
	inst.Name = name
}

// --- [ phi ] -----------------------------------------------------------------

// InstPhi represents a phi instruction.
//
// References:
//    http://llvm.org/docs/LangRef.html#phi-instruction
type InstPhi struct {
	// Name of the local variable associated with the instruction.
	Name string
	// Type of the instruction.
	Type Type
	// Incoming values.
	Incs []*Incoming
	// Metadata attached to the instruction.
	Metadata []*AttachedMD
}

// GetName returns the name of the value.
func (inst *InstPhi) GetName() string {
	return inst.Name
}

// SetName sets the name of the value.
func (inst *InstPhi) SetName(name string) {
	inst.Name = name
}

// Incoming represents an incoming value of a phi instruction.
type Incoming struct {
	// Incoming value.
	X Value
	// Predecessor basic block of the incoming value.
	Pred NamedValue
}

// --- [ select ] --------------------------------------------------------------

// InstSelect represents a select instruction.
//
// References:
//    http://llvm.org/docs/LangRef.html#select-instruction
type InstSelect struct {
	// Name of the local variable associated with the instruction.
	Name string
	// Selection condition.
	Cond Value
	// Operands.
	X, Y Value
	// Metadata attached to the instruction.
	Metadata []*AttachedMD
}

// GetName returns the name of the value.
func (inst *InstSelect) GetName() string {
	return inst.Name
}

// SetName sets the name of the value.
func (inst *InstSelect) SetName(name string) {
	inst.Name = name
}

// --- [ call ] ----------------------------------------------------------------

// InstCall represents a call instruction.
//
// References:
//    http://llvm.org/docs/LangRef.html#call-instruction
type InstCall struct {
	// Name of the local variable associated with the instruction.
	Name string
	// Type of the instruction; or callee type signature.
	Type Type
	// Callee.
	Callee Value
	// Function arguments.
	Args []Value
	// Calling convention.
	CallConv CallConv
	// Metadata attached to the instruction.
	Metadata []*AttachedMD
}

// GetName returns the name of the value.
func (inst *InstCall) GetName() string {
	return inst.Name
}

// SetName sets the name of the value.
func (inst *InstCall) SetName(name string) {
	inst.Name = name
}

// --- [ va_arg ] --------------------------------------------------------------

// --- [ landingpad ] ----------------------------------------------------------

// --- [ catchpad ] ------------------------------------------------------------

// --- [ cleanuppad ] ----------------------------------------------------------

// isValue ensures that only values can be assigned to the ast.Value interface.
func (*InstICmp) isValue()   {}
func (*InstFCmp) isValue()   {}
func (*InstPhi) isValue()    {}
func (*InstSelect) isValue() {}
func (*InstCall) isValue()   {}

// isInst ensures that only instructions can be assigned to the ast.Instruction
// interface.
func (*InstICmp) isInst()   {}
func (*InstFCmp) isInst()   {}
func (*InstPhi) isInst()    {}
func (*InstSelect) isInst() {}
func (*InstCall) isInst()   {}
