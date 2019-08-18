package parser

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"io/ioutil"
	"reflect"
	"strings"
)

// Type is the type defenition struct
type Type struct {
	FileName string
	Name     string
	Fields   []Field
	Package  string
}

// Field encapsulates the struct field metadata needed for the template execution
type Field struct {
	Name      string
	Type      string
	EnvTag    string
	IsPointer bool
	IsArray   bool
}

// NewType returns a new instance of Type with given name
func NewType(name string) *Type {
	return &Type{
		Name: name,
	}
}

// getStruct returns the struct metadata if the given node is a struct
func getStruct(nodeType ast.Node) *ast.StructType {
	switch node := nodeType.(type) {
	case *ast.StructType:
		return node
	default:
		return nil
	}
}

// Parse parses Type metadata from given file using go parser & ast
func (t *Type) Parse(fileName string) error {
	t.FileName = fileName
	fset := token.NewFileSet()
	content, err := ioutil.ReadFile(fileName)
	if err != nil {
		return err
	}
	f, err := parser.ParseFile(fset, fileName, content, parser.ParseComments)
	if err != nil {
		return err
	}
	// Look up the AST
	ast.Inspect(f, func(node ast.Node) bool {
		switch nodeType := node.(type) {
		case *ast.TypeSpec:
			if node := getStruct(nodeType.Type); node != nil {
				// Fetch info for only the given struct
				if t.Name == nodeType.Name.String() {
					// Helper to populate struct's field and tags info
					t.Package = f.Name.String()
					t.Fields = getFields(node)
					return false
				}
			}
		}
		return true
	})
	return nil
}

// getFields will transforms the field metadata returned by go ast to the template's format
func getFields(node *ast.StructType) []Field {
	var fields []Field
	for _, field := range node.Fields.List {
		var tags reflect.StructTag
		if field.Tag != nil {
			tags = reflect.StructTag(strings.Trim(field.Tag.Value, "`"))
		}
		if len(field.Names) == 0 {
			fieldType := types.ExprString(field.Type)
			fields = append(fields, Field{
				Name:      "",
				Type:      cleanTypeStr(fieldType),
				EnvTag:    getEnvSourceTag(tags, fieldType),
				IsPointer: isPointer(fieldType),
				IsArray:   isArray(fieldType),
			})
			continue
		}
		for _, fieldName := range field.Names {
			fieldType := types.ExprString(field.Type)
			fields = append(fields, Field{
				Name:      fieldName.Name,
				Type:      cleanTypeStr(fieldType),
				IsPointer: isPointer(fieldType),
				IsArray:   isArray(fieldType),
				EnvTag:    getEnvSourceTag(tags, fieldName.Name),
			})
		}
	}
	return fields
}

// isPointer checks if a given type is a pointer or not
func isPointer(typeName string) bool {
	return len(typeName) > 0 && typeName[0] == '*'
}

// isArray checks is a given type is an array or not
func isArray(typeName string) bool {
	return len(typeName) > 2 && typeName[:2] == "[]"
}

// getEnvSourceTag will says what is ths env variable to be used to fetch the data
func getEnvSourceTag(tags reflect.StructTag, fieldName string) string {
	tag, ok := tags.Lookup("env")
	if !ok {
		return strings.ToUpper(fieldName)
	}
	return tag
}

// cleanTypeStr will strip all unwanted space and other characters to return the type name
func cleanTypeStr(typ string) string {
	typ = strings.TrimSpace(typ)
	typ = strings.TrimLeft(typ, "*")
	typ = strings.TrimLeft(typ, "[]")
	return typ
}
