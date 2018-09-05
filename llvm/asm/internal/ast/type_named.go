package ast

// NamedType represents the type definition of a type alias or an identified
// struct type.
type NamedType struct {
	// Type name.
	Name string
	// Type definition.
	Def Type
}

// GetName returns the name of the type.
func (t *NamedType) GetName() string {
	return t.Name
}

// isType ensures that only types can be assigned to the ast.Type interface.
func (*NamedType) isType() {}

// NamedTypeDummy represents a dummy type identifier.
type NamedTypeDummy struct {
	// Type name.
	Name string
}

// isType ensures that only types can be assigned to the ast.Type interface.
func (*NamedTypeDummy) isType() {}
