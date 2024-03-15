package main

import (
	"errors"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConsumes(t *testing.T) {
	tests := []struct {
		desc     string
		a        string
		expected []string
	}{
		{
			desc:     "accept json, returns json",
			a:        "json",
			expected: []string{ajson},
		},
		{
			desc:     "accept xml, returns xml",
			a:        "xml",
			expected: []string{axml},
		},
		{
			desc:     "accept plain, returns plain",
			a:        "plain",
			expected: []string{aplain},
		},
		{
			desc:     "accept html, returns html",
			a:        "html",
			expected: []string{ahtml},
		},
		{
			desc:     "accept thrift_binary, returns thrift binary",
			a:        "thrift_binary",
			expected: []string{content_type_thrift_binary},
		},
		{
			desc:     "accept thrift_json, returns thrift json",
			a:        "thrift_json",
			expected: []string{content_type_thrift_json},
		},
		{
			desc:     "accept thrift_webcontent_binary, returns thrift web content binary",
			a:        "thrift_webcontent_binary",
			expected: []string{content_type_thrift_binary_webcontent_v1},
		},
		{
			desc:     "accept thrift_webcontent_json, returns thrift web content json",
			a:        "thrift_webcontent_json",
			expected: []string{content_type_thrift_json_webcontent_v1},
		},
		{
			desc:     "accept multipart, returns multipart/form-data",
			a:        "multipart",
			expected: []string{contentTypeMultipartFormData},
		},
		{
			desc:     "accept form, returns application/x-www-form-urlencoded",
			a:        "form",
			expected: []string{contentTypeFormURLEncoded},
		},
		{
			desc:     "unknown accept, returns empty slice",
			a:        "skippy-chunky",
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			actual := consumes(tt.a)
			assert.Equal(t, tt.expected, actual)
		})
	}
}

func TestGetGoFilesInPackage(t *testing.T) {
	tests := []struct {
		desc      string
		pkg       string
		mockGetwd func() (dir string, err error)
		expected  map[string]*ast.Package
		isError   assert.ErrorAssertionFunc
	}{
		{
			desc: "Test package is not ignored, return list of parsed go files",
			pkg:  "testdata/router",
			mockGetwd: func() (string, error) {
				return "/a/test/root/package", nil
			},
			expected: func() map[string]*ast.Package {
				pkg := "testdata/router"
				pkgs, err := parser.
					ParseDir(token.NewFileSet(), pkg, func(info os.FileInfo) bool {
						return true
					}, parser.ParseComments)

				assert.NoError(t, err)

				return pkgs
			}(),
			isError: assert.NoError,
		},
		{
			desc: "Test package is ignored, returns nil without error",
			pkg:  "/a/test/root/package/handlers",
			mockGetwd: func() (string, error) {
				return "/a/test/root/package", nil
			},
			expected: nil,
			isError:  assert.NoError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			getwd = tt.mockGetwd
			actual, err := getGoFilesInPackage(tt.pkg)
			assert.Equal(t, tt.expected, actual)
			tt.isError(t, err)
		})
	}
}

func TestIsPackageIgnored(t *testing.T) {
	tests := []struct {
		desc      string
		pkg       string
		mockGetwd func() (dir string, err error)
		expected  bool
	}{
		{
			desc: "Test Getwd returns error, returns false",
			pkg:  "",
			mockGetwd: func() (string, error) {
				return "", errors.New("error getting current dir")
			},
			expected: false,
		},
		{
			desc: "Test package is not ignored, returns false",
			pkg:  "/a/test/root/package/notignored",
			mockGetwd: func() (string, error) {
				return "/a/test/root/package", nil
			},
			expected: false,
		},
		{
			desc: "Test package is ignored, returns true",
			pkg:  "/a/test/root/package/handlers",
			mockGetwd: func() (string, error) {
				return "/a/test/root/package", nil
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			getwd = tt.mockGetwd
			actual := isPackageIgnored(tt.pkg)
			assert.Equal(t, tt.expected, actual)
		})
	}
}

func TestGetProjectFromImportPath(t *testing.T) {
	tests := []struct {
		desc     string
		path     string
		expected string
		isError  assert.ErrorAssertionFunc
	}{
		{
			desc:     "Input path is not an assumed path returns error",
			path:     "go.uber.org/zap",
			expected: "",
			isError:  assert.Error,
		},
		{
			desc:     "Input path is an assumed path returns the project name",
			path:     "github.com/uber/zap",
			expected: "zap",
			isError:  assert.NoError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			actual, err := getProjectFromImportPath(tt.path)
			assert.Equal(t, tt.expected, actual)
			tt.isError(t, err)
		})
	}
}
