package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"

	"github.com/astaxie/beego/swagger"
	"github.com/stretchr/testify/assert"
	"github.com/zalora/bee/testdata"
)

func TestConstructObjectPropertie(t *testing.T) {
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
			field:       &testdata.StructRepresentation,
			realTypes:   []string{},
			pathInfo:    map[string]string{},
			expected: swagger.Propertie{
				Type: "object",
				Properties: map[string]swagger.Propertie{
					"field_name": {
						Type: "string",
					},
					"fieldName2": {
						Type:   "integer",
						Format: "int64",
					},
					"expandedField1": {
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
				Value: &testdata.StructRepresentation,
			}, // map[int64]structName{}
			realTypes:         []string{},
			pathInfo:          map[string]string{},
			expected:          swagger.Propertie{},
			expectedRealTypes: []string{},
		},
		{
			description: "map type",
			field: &ast.MapType{
				Key: &ast.Ident{
					Name: "string",
				},
				Value: &testdata.StructRepresentation,
			}, // map[string]structName{}
			realTypes: []string{},
			pathInfo:  map[string]string{},
			expected: swagger.Propertie{
				Type: "object",
				AdditionalProperties: &swagger.Propertie{
					Type: "object",
					Properties: map[string]swagger.Propertie{
						"field_name": {
							Type: "string",
						},
						"fieldName2": {
							Type:   "integer",
							Format: "int64",
						},
						"expandedField1": {
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
			}, // package.object()
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

func TestGeneratePathInfo(t *testing.T) {
	tests := []struct {
		description string
		src         string
		expected    map[string]string
		isError     assert.ErrorAssertionFunc
	}{
		{
			description: "no imported package",
			src: string(`
			package main

			func main() {
				a := 1 + 1
			}
			`),
			expected: map[string]string{},
			isError:  assert.NoError,
		},
		{
			description: "non internal imported packages",
			src: string(`
			package main

			import (
				"fmt"
				"encoding/json"
			)

			func main() {
				a := 1 + 1
			}
			`),
			expected: map[string]string{},
			isError:  assert.NoError,
		},
		{
			description: "internal imported packages",
			src: string(`
			package main

			import (
				"github.com/zalora/bee/packageA"
				"github.com/zalora/bee/packageB/packageBA"
			)

			func main() {
				a := 1 + 1
			}
			`),
			expected: map[string]string{
				"packageA":  "/packageA",
				"packageBA": "/packageB/packageBA",
			},
			isError: assert.NoError,
		},
		{
			description: "with named internal package",
			src: string(`
			package main

			import (
				"github.com/zalora/bee/packageA"
				packageC "github.com/zalora/bee/packageB/packageBA"
			)

			func main() {
				a := 1 + 1
			}
			`),
			expected: map[string]string{
				"packageA": "/packageA",
				"packageC": "/packageB/packageBA",
			},
			isError: assert.NoError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			f, err := parser.ParseFile(token.NewFileSet(), "", tt.src, 0)
			if err != nil {
				t.Errorf("parser.ParseFile() invalid src = %v", tt.src)
			}

			sub, err := generatePathInfo(f)
			assert.Equal(t, tt.expected, sub)
			tt.isError(t, err)
		})
	}
}

func TestParseObject(t *testing.T) {
	tests := []struct {
		description       string
		res               *objectResource
		expectedSchema    *swagger.Schema
		expectedRealTypes *[]string
	}{
		{
			description: "parse struct",
			res: &objectResource{
				object: &ast.Object{
					Kind: ast.Typ,
					Name: "structObject",
					Decl: &ast.TypeSpec{
						Name: &ast.Ident{
							Name: "structObject",
						},
						Type: &testdata.StructRepresentation,
					},
				},
				schema:    &swagger.Schema{},
				realTypes: &[]string{},
			},
			expectedSchema: &swagger.Schema{
				Title: "structObject",
				Type:  "object",
				Properties: map[string]swagger.Propertie{
					"field_name": {
						Type: "string",
					},
					"fieldName2": {
						Type:   "integer",
						Format: "int64",
					},
					"expandedField1": {
						Type:   "integer",
						Format: "int64",
					},
				},
			},
			expectedRealTypes: &[]string{},
		},
		{
			description: "identifier type",
			res: &objectResource{
				object: &ast.Object{
					Kind: ast.Typ,
					Name: "anyType",
					Decl: &ast.TypeSpec{
						Name: &ast.Ident{
							Name: "anyType",
						},
						Type: &ast.Ident{
							Name: "int64",
						},
					},
				}, // type anyType int64
				schema:    &swagger.Schema{},
				realTypes: &[]string{},
			},
			expectedSchema: &swagger.Schema{
				Title:  "anyType",
				Type:   "integer",
				Format: "int64",
			},
			expectedRealTypes: &[]string{},
		},
		{
			description: "unhandled type",
			res: &objectResource{
				object: &ast.Object{
					Kind: ast.Typ,
					Name: "anyType",
					Decl: &ast.TypeSpec{
						Name: &ast.Ident{
							Name: "anyType",
						},
						Type: &ast.FuncType{},
					},
				}, // type anyType int64
				schema:    &swagger.Schema{},
				realTypes: &[]string{},
			},
			expectedSchema:    &swagger.Schema{},
			expectedRealTypes: &[]string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			tt.res.parseObject()

			assert.Equal(t, tt.expectedSchema, tt.res.schema)
			assert.Equal(t, tt.expectedRealTypes, tt.res.realTypes)
		})
	}
}

func TestDeepCopy(t *testing.T) {
	tests := []struct {
		description    string
		value          interface{}
		expectedResult interface{}
		isError        assert.ErrorAssertionFunc
	}{
		{
			description:    "nil value",
			value:          nil,
			expectedResult: nil,
			isError:        assert.NoError,
		},
		{
			description: "struct",
			value: swagger.Item{
				Ref: "Reference#1",
				Get: &swagger.Operation{
					Description: "A Get Method",
					OperationID: "#1",
				},
			},
			expectedResult: swagger.Item{
				Ref: "Reference#1",
				Get: &swagger.Operation{
					Description: "A Get Method",
					OperationID: "#1",
				},
			},
			isError: assert.NoError,
		},
		{
			description: "pointer struct",
			value: &swagger.Item{
				Ref: "Reference#2",
				Get: &swagger.Operation{
					Description: "A Get Method",
					OperationID: "#2",
				},
			},
			expectedResult: &swagger.Item{
				Ref: "Reference#2",
				Get: &swagger.Operation{
					Description: "A Get Method",
					OperationID: "#2",
				},
			},
			isError: assert.NoError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			sub, err := deepCopy(tt.value)

			assert.Equal(t, tt.expectedResult, sub)
			assert.NotSame(t, tt.expectedResult, sub)
			tt.isError(t, err)
		})
	}
}

func TestReplicateSwaggerItem(t *testing.T) {
	tests := []struct {
		description  string
		item         *swagger.Item
		expectedItem *swagger.Item
		isError      assert.ErrorAssertionFunc
	}{
		{
			description:  "empty swagger item",
			item:         &swagger.Item{},
			expectedItem: &swagger.Item{},
			isError:      assert.NoError,
		},
		{
			description: "non-empty swagger item",
			item: &swagger.Item{
				Ref: "Reference#1",
				Post: &swagger.Operation{
					Tags:        []string{"Tag#1"},
					OperationID: "OperationID#1",
				},
			},
			expectedItem: &swagger.Item{
				Ref: "Reference#1",
				Post: &swagger.Operation{
					Tags:        []string{"Tag#1"},
					OperationID: "OperationID#1",
				},
			},
			isError: assert.NoError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			sub, err := replicateSwaggerItem(tt.item)

			assert.Equal(t, tt.expectedItem, sub)
			assert.NotSame(t, tt.expectedItem, sub)
			tt.isError(t, err)
		})
	}
}

func TestEnrichItem(t *testing.T) {
	tests := []struct {
		description  string
		item         *swagger.Item
		tag          string
		route        string
		expectedItem *swagger.Item
	}{
		{
			description: "minimum one method is is used by an endpoint",
			item: &swagger.Item{
				Ref: "Reference#1",
				Get: &swagger.Operation{
					OperationID: "Controller.Get Update Cart",
				},
			},
			tag:   "cart",
			route: "/cart",
			expectedItem: &swagger.Item{
				Ref: "Reference#1",
				Get: &swagger.Operation{
					Tags:        []string{"cart"},
					OperationID: "cart.Controller.Get Update Cart",
				},
			},
		},
		{
			description: "no method is used by any endpoint",
			item: &swagger.Item{
				Ref: "Reference#1",
			},
			tag:   "cart",
			route: "/cart",
			expectedItem: &swagger.Item{
				Ref: "Reference#1",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			enrichItem(tt.item, tt.tag, tt.route)

			assert.Equal(t, tt.expectedItem, tt.item)
		})
	}
}

func TestOperationIDFormat(t *testing.T) {
	tests := []struct {
		description   string
		value         string
		expectedValue string
	}{
		{
			description:   "already in operationID format",
			value:         "post.endpoint",
			expectedValue: "post.endpoint",
		},
		{
			description:   "value with trailing slash",
			value:         "post/endpoint/",
			expectedValue: "post.endpoint",
		},
		{
			description:   "value with leading and trailing slash",
			value:         "/post/endpoint/",
			expectedValue: "post.endpoint",
		},
	}

	for _, tt := range tests {
		sub := operationIDFormat(tt.value)

		assert.Equal(t, tt.expectedValue, sub)
	}
}
