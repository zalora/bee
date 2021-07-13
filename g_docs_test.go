package main

import (
	"go/ast"
	"testing"

	"github.com/astaxie/beego/swagger"
	"github.com/stretchr/testify/assert"
)

var structAstRepresentation = ast.StructType{
	Fields: &ast.FieldList{
		List: []*ast.Field{
			{
				Names: []*ast.Ident{
					{
						Name: "fieldName",
						Obj: &ast.Object{
							Kind: ast.Var,
							Name: "fieldName",
							Decl: ast.Field{
								Names: []*ast.Ident{
									{
										Name: "fieldName",
									},
								},
								Type: &ast.Ident{
									Name: "string",
								},
							},
						},
					},
				},
				Type: &ast.Ident{
					Name: "string",
				},
			},
			{
				Names: []*ast.Ident{
					{
						Name: "fieldName2",
						Obj: &ast.Object{
							Kind: ast.Var,
							Name: "fieldName2",
							Decl: ast.Field{
								Names: []*ast.Ident{
									{
										Name: "fieldName2",
									},
								},
								Type: &ast.Ident{
									Name: "int64",
								},
							},
						},
					},
				},
				Type: &ast.Ident{
					Name: "int64",
				},
			},
		},
	},
} // type structName struct { fieldName string; fieldName2 int64; }

func Test_ConstructObjectPropertie(t *testing.T) {
	tests := []struct {
		description       string
		field             ast.Expr
		packageName       string
		realTypes         []string
		pathInfo          map[string]string
		expected          swagger.Propertie
		expectedRealTypes []string
	}{
		{
			description: "basic golang type",
			field: &ast.Ident{
				Name: "int64",
			}, // int64
			realTypes: []string{},
			pathInfo:  map[string]string{},
			expected: swagger.Propertie{
				Type:   "integer",
				Format: "int64",
			},
			expectedRealTypes: []string{},
		},
		{
			description: "star expression",
			field: &ast.StarExpr{
				X: &ast.Ident{
					Name: "maskedInt64",
				},
			}, // type maskedInt64 int64; maskedInt64 := *123;
			packageName: "package",
			realTypes:   []string{},
			pathInfo: map[string]string{
				"package": "package",
			},
			expected: swagger.Propertie{
				Ref: "#/definitions/package.maskedInt64",
			},
			expectedRealTypes: []string{"package.maskedInt64"},
		},
		{
			description: "array type",
			field: &ast.ArrayType{
				Elt: &ast.Ident{
					Name: "string",
				},
			}, // []string{}
			realTypes: []string{},
			pathInfo:  map[string]string{},
			expected: swagger.Propertie{
				Type: "array",
				Items: &swagger.Propertie{
					Type: "string",
				},
			},
			expectedRealTypes: []string{},
		},
		{
			description: "struct type",
			field:       &structAstRepresentation,
			realTypes:   []string{},
			pathInfo:    map[string]string{},
			expected: swagger.Propertie{
				Type: "object",
				Properties: map[string]swagger.Propertie{
					"fieldName": {
						Type: "string",
					},
					"fieldName2": {
						Type:   "integer",
						Format: "int64",
					},
				},
			},
			expectedRealTypes: []string{},
		},
		{
			description: "map type with non string key",
			field: &ast.MapType{
				Key: &ast.Ident{
					Name: "int64",
				},
				Value: &structAstRepresentation,
			},
			realTypes:         []string{},
			pathInfo:          map[string]string{},
			expected:          swagger.Propertie{},
			expectedRealTypes: []string{},
		},
		{
			description: "map type",
			field: &ast.MapType{
				Key: &ast.Ident{
					Name: "string", // TODO: map with non-string key
				},
				Value: &structAstRepresentation,
			},
			realTypes: []string{},
			pathInfo:  map[string]string{},
			expected: swagger.Propertie{
				Type: "object",
				AdditionalProperties: &swagger.Propertie{
					Type: "object",
					Properties: map[string]swagger.Propertie{
						"fieldName": {
							Type: "string",
						},
						"fieldName2": {
							Type:   "integer",
							Format: "int64",
						},
					},
				},
			},
			expectedRealTypes: []string{},
		},
		{
			description: "identifier type",
			field: &ast.Ident{
				Name: "anyType",
				Obj: &ast.Object{
					Kind: ast.Typ,
					Name: "",
					Decl: &ast.TypeSpec{
						Name: &ast.Ident{
							Name: "anyType",
						},
						Type: &ast.Ident{
							Name: "int64",
						},
					},
				},
			}, // type anyType int64
			realTypes: []string{},
			pathInfo:  map[string]string{},
			expected: swagger.Propertie{
				Type:   "integer",
				Format: "int64",
			},
			expectedRealTypes: []string{},
		},
		{
			description: "selector expression",
			field: &ast.SelectorExpr{
				X: &ast.Ident{
					Name: "package",
				},
				Sel: &ast.Ident{
					Name: "object",
				},
			},
			realTypes: []string{},
			pathInfo: map[string]string{
				"package": "outerpackage/package",
			},
			expected: swagger.Propertie{
				Ref: "#/definitions/package.object",
			},
			expectedRealTypes: []string{
				"outerpackage.package.object",
			},
		},
		{
			description:       "type is not handled",
			field:             nil,
			realTypes:         []string{},
			pathInfo:          map[string]string{},
			expected:          swagger.Propertie{},
			expectedRealTypes: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			sub := constructObjectPropertie(tt.field, tt.packageName, &tt.realTypes, tt.pathInfo)

			assert.Equal(t, tt.expected, sub)
			assert.Equal(t, tt.expectedRealTypes, tt.realTypes)
		})
	}
}
