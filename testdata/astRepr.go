package testdata

import (
	"go/ast"
	"go/token"
)

// StructRepresentation is an ast struct representation used in test files
// this represents
// type aStruct struct {
//     fieldName  string `json:"field_name",omitempty`
//     fieldName2 int64
//     expandedField
// },
var StructRepresentation = ast.StructType{
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
				Tag: &ast.BasicLit{
					Kind:  token.STRING,
					Value: "json:\"field_name\",omitempty",
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
			}, // field without a tag
			{
				Type: &ast.StructType{
					Fields: &ast.FieldList{
						List: []*ast.Field{
							{
								Names: []*ast.Ident{
									{
										Name: "expandedField1",
										Obj: &ast.Object{
											Kind: ast.Var,
											Name: "expandedField1",
											Decl: ast.Field{
												Names: []*ast.Ident{
													{
														Name: "expandedField1",
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
				},
			}, // unnamed field
		},
	},
} // type structName struct { fieldName string; fieldName2 int64; }
