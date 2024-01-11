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
