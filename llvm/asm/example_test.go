package asm_test

import (
	"log"

	"github.com/kr/pretty"
	"github.com/geode-lang/geode/llvm/asm"
)

func Example() {
	// Parse the LLVM IR assembly file `rand.ll`.
	m, err := asm.ParseFile("testdata/rand.ll")
	if err != nil {
		log.Fatal(err)
	}
	// Pretty-print the data types of the parsed LLVM IR module.
	pretty.Println(m)
	// Output:
	//
	// &ir.Module{
	//     DataLayout:   "",
	//     TargetTriple: "",
	//     Types:        nil,
	//     Globals:      {
	//         &ir.Global{
	//             Name: "seed",
	//             Typ:  &types.PointerType{
	//                 Name:      "",
	//                 Elem:      &types.IntType{Name:"", Size:32},
	//                 AddrSpace: 0,
	//             },
	//             Content: &types.IntType{Name:"", Size:32},
	//             Init:    &constant.Int{
	//                 Typ: &types.IntType{(CYCLIC REFERENCE)},
	//                 X:   &big.Int{},
	//             },
	//             IsConst:  false,
	//             Metadata: {
	//             },
	//         },
	//     },
	//     Funcs: {
	//         &ir.Function{
	//             Parent: &ir.Module{(CYCLIC REFERENCE)},
	//             Name:   "abs",
	//             Typ:    &types.PointerType{
	//                 Name: "",
	//                 Elem: &types.FuncType{
	//                     Name:   "",
	//                     Ret:    &types.IntType{Name:"", Size:32},
	//                     Params: {
	//                         &types.Param{
	//                             Name: "x",
	//                             Typ:  &types.IntType{Name:"", Size:32},
	//                         },
	//                     },
	//                     Variadic: false,
	//                 },
	//                 AddrSpace: 0,
	//             },
	//             Sig: &types.FuncType{
	//                 Name:   "",
	//                 Ret:    &types.IntType{Name:"", Size:32},
	//                 Params: {
	//                     &types.Param{
	//                         Name: "x",
	//                         Typ:  &types.IntType{Name:"", Size:32},
	//                     },
	//                 },
	//                 Variadic: false,
	//             },
	//             CallConv: 0x0,
	//             Blocks:   nil,
	//             Metadata: {
	//             },
	//             mu: sync.Mutex{},
	//         },
	//         &ir.Function{
	//             Parent: &ir.Module{(CYCLIC REFERENCE)},
	//             Name:   "rand",
	//             Typ:    &types.PointerType{
	//                 Name: "",
	//                 Elem: &types.FuncType{
	//                     Name:   "",
	//                     Ret:    &types.IntType{Name:"", Size:32},
	//                     Params: {
	//                     },
	//                     Variadic: false,
	//                 },
	//                 AddrSpace: 0,
	//             },
	//             Sig: &types.FuncType{
	//                 Name:   "",
	//                 Ret:    &types.IntType{Name:"", Size:32},
	//                 Params: {
	//                 },
	//                 Variadic: false,
	//             },
	//             CallConv: 0x0,
	//             Blocks:   {
	//                 &ir.BasicBlock{
	//                     Parent: &ir.Function{(CYCLIC REFERENCE)},
	//                     Name:   "0",
	//                     Insts:  {
	//                         &ir.InstLoad{
	//                             Parent:   &ir.BasicBlock{(CYCLIC REFERENCE)},
	//                             Name:     "1",
	//                             Typ:      &types.IntType{(CYCLIC REFERENCE)},
	//                             Src:      &ir.Global{(CYCLIC REFERENCE)},
	//                             Metadata: {
	//                             },
	//                         },
	//                         &ir.InstMul{
	//                             Parent: &ir.BasicBlock{(CYCLIC REFERENCE)},
	//                             Name:   "2",
	//                             X:      &ir.InstLoad{(CYCLIC REFERENCE)},
	//                             Y:      &constant.Int{
	//                                 Typ: &types.IntType{Name:"", Size:32},
	//                                 X:   &big.Int{
	//                                     neg: false,
	//                                     abs: {0x15a4e35},
	//                                 },
	//                             },
	//                             Metadata: {
	//                             },
	//                         },
	//                         &ir.InstAdd{
	//                             Parent: &ir.BasicBlock{(CYCLIC REFERENCE)},
	//                             Name:   "3",
	//                             X:      &ir.InstMul{(CYCLIC REFERENCE)},
	//                             Y:      &constant.Int{
	//                                 Typ: &types.IntType{Name:"", Size:32},
	//                                 X:   &big.Int{
	//                                     neg: false,
	//                                     abs: {0x1},
	//                                 },
	//                             },
	//                             Metadata: {
	//                             },
	//                         },
	//                         &ir.InstStore{
	//                             Parent:   &ir.BasicBlock{(CYCLIC REFERENCE)},
	//                             Src:      &ir.InstAdd{(CYCLIC REFERENCE)},
	//                             Dst:      &ir.Global{(CYCLIC REFERENCE)},
	//                             Metadata: {
	//                             },
	//                         },
	//                         &ir.InstCall{
	//                             Parent: &ir.BasicBlock{(CYCLIC REFERENCE)},
	//                             Name:   "4",
	//                             Callee: &ir.Function{(CYCLIC REFERENCE)},
	//                             Sig:    &types.FuncType{(CYCLIC REFERENCE)},
	//                             Args:   {
	//                                 &ir.InstAdd{(CYCLIC REFERENCE)},
	//                             },
	//                             CallConv: 0x0,
	//                             Metadata: {
	//                             },
	//                         },
	//                     },
	//                     Term: &ir.TermRet{
	//                         Parent: &ir.BasicBlock{(CYCLIC REFERENCE)},
	//                         X:      &ir.InstCall{
	//                             Parent: &ir.BasicBlock{(CYCLIC REFERENCE)},
	//                             Name:   "4",
	//                             Callee: &ir.Function{(CYCLIC REFERENCE)},
	//                             Sig:    &types.FuncType{(CYCLIC REFERENCE)},
	//                             Args:   {
	//                                 &ir.InstAdd{(CYCLIC REFERENCE)},
	//                             },
	//                             CallConv: 0x0,
	//                             Metadata: {
	//                             },
	//                         },
	//                         Metadata: {
	//                         },
	//                     },
	//                 },
	//             },
	//             Metadata: {
	//             },
	//             mu: sync.Mutex{},
	//         },
	//     },
	//     NamedMetadata: nil,
	//     Metadata:      nil,
	// }
}
