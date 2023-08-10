package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"

	"github.com/astaxie/beego/swagger"
	"github.com/stretchr/testify/assert"
)

func TestGenerateChiTags(t *testing.T) {
	tests := []struct {
		desc     string
		params   func() (*ast.File, *token.FileSet)
		expected []swagger.Tag
	}{
		{
			desc: "Test success generate chi tags",
			params: func() (*ast.File, *token.FileSet) {
				code := []byte(`
				package router
				
				import (
					"net/http"
				
					"github.com/companyname/appname/handlers/testendpoint"
					"github.com/go-chi/chi/v5"
				)
				
				func New() {
					// Application endpoints
					mux := chi.NewRouter()
				
					// Some comments explaining why do we do this.
					mux.NotFound(func(w http.ResponseWriter, r *http.Request) {
						w.WriteHeader(http.StatusNotFound)
						_, _ = w.Write([]byte(""))
					})
				
					mux.Group(func(r chi.Router) {
						r.Route("/v1", func(r chi.Router) {
						
							// Test API
							r.Route("/testendpoint", func(r chi.Router) {
								r.Get("/", testendpoint.Get)
							})
						})
					})
				
				}
				`)

				fset := token.NewFileSet()
				f, err := parser.ParseFile(fset, "", code, parser.ParseComments)
				assert.NoError(t, err)

				return f, fset
			},
			expected: []swagger.Tag{
				{
					Name:         "testendpoint",
					Description:  "Test API\n",
					ExternalDocs: nil,
				},
			},
		},
		{
			desc: "Test tag is not defined, return nil tag",
			params: func() (*ast.File, *token.FileSet) {
				code := []byte(`
				package router
				
				import (
					"net/http"
				
					"github.com/companyname/appname/handlers/testendpoint"
					"github.com/go-chi/chi/v5"
				)
				
				func New() {
					// Application endpoints
					mux := chi.NewRouter()
				
					// Some comments explaining why do we do this.
					mux.NotFound(func(w http.ResponseWriter, r *http.Request) {
						w.WriteHeader(http.StatusNotFound)
						_, _ = w.Write([]byte(""))
					})
				
					mux.Group(func(r chi.Router) {
						r.Route("/v1", func(r chi.Router) {
							r.Route("/testendpoint", func(r chi.Router) {
								r.Get("/", testendpoint.Get)
							})
						})
					})
				
				}
				`)

				fset := token.NewFileSet()
				f, err := parser.ParseFile(fset, "", code, parser.ParseComments)
				assert.NoError(t, err)

				return f, fset
			},
			expected: nil,
		},
		{
			desc: "Test function New() is not present, no tag is returned",
			params: func() (*ast.File, *token.FileSet) {
				code := []byte(`
				package router
				
				import (
					"net/http"
				
					"github.com/companyname/appname/handlers/testendpoint"
					"github.com/go-chi/chi/v5"
				)
				
				func Old() {
					// Application endpoints
					mux := chi.NewRouter()
				
					// Some comments explaining why do we do this.
					mux.NotFound(func(w http.ResponseWriter, r *http.Request) {
						w.WriteHeader(http.StatusNotFound)
						_, _ = w.Write([]byte(""))
					})
				
					mux.Group(func(r chi.Router) {
						r.Route("/v1", func(r chi.Router) {
							r.Route("/testendpoint", func(r chi.Router) {
								r.Get("/", testendpoint.Get)
							})
						})
					})
				
				}
				`)

				fset := token.NewFileSet()
				f, err := parser.ParseFile(fset, "", code, parser.ParseComments)
				assert.NoError(t, err)

				return f, fset
			},
			expected: nil,
		},
		{
			desc: "Test route call is not present, return nil tag",
			params: func() (*ast.File, *token.FileSet) {
				code := []byte(`
				package router
				
				import (
					"net/http"
				
					"github.com/companyname/appname/handlers/testendpoint"
					"github.com/go-chi/chi/v5"
				)
				
				func New() {
					// Application endpoints
					mux := chi.NewRouter()
				
					// Some comments explaining why do we do this.
					mux.NotFound(func(w http.ResponseWriter, r *http.Request) {
						w.WriteHeader(http.StatusNotFound)
						_, _ = w.Write([]byte(""))
					})
				
					mux.Group(func(r chi.Router) {
					})
				
				}
				`)

				fset := token.NewFileSet()
				f, err := parser.ParseFile(fset, "", code, parser.ParseComments)
				assert.NoError(t, err)

				return f, fset
			},
			expected: nil,
		},
		{
			desc: "Test mux.Group call is not present, return nil tag",
			params: func() (*ast.File, *token.FileSet) {
				code := []byte(`
				package router
				
				import (
					"net/http"
				
					"github.com/companyname/appname/handlers/testendpoint"
					"github.com/go-chi/chi/v5"
				)
				
				func New() {
					// Application endpoints
					mux := chi.NewRouter()
				
					// Some comments explaining why do we do this.
					mux.NotFound(func(w http.ResponseWriter, r *http.Request) {
						w.WriteHeader(http.StatusNotFound)
						_, _ = w.Write([]byte(""))
					})	
				}
				`)

				fset := token.NewFileSet()
				f, err := parser.ParseFile(fset, "", code, parser.ParseComments)
				assert.NoError(t, err)

				return f, fset
			},
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			actual := generateChiTags(tt.params())
			assert.Equal(t, tt.expected, actual)
		})
	}
}
