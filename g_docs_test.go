package main

import (
	"errors"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type mockOS struct {
	mock.Mock
}

func (m *mockOS) Getwd() (string, error) {
	args := m.Called()
	err := args.Error(1)
	if err != nil {
		return "", err
	}

	return args.Get(0).(string), nil
}

func TestGetGoFilesInPackage(t *testing.T) {
	tests := []struct {
		desc     string
		pkg      string
		mockOS   *mockOS
		expected map[string]*ast.Package
		isError  assert.ErrorAssertionFunc
	}{
		{
			desc: "Test package is not ignored, return list of parsed go files",
			pkg:  "testdata/router",
			mockOS: func() *mockOS {
				m := new(mockOS)

				m.On("Getwd").Return("/a/test/root/package", nil)
				return m
			}(),
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
			mockOS: func() *mockOS {
				m := new(mockOS)

				m.On("Getwd").Return("/a/test/root/package", nil)
				return m
			}(),
			expected: nil,
			isError:  assert.NoError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			defaultOS = tt.mockOS
			actual, err := getGoFilesInPackage(tt.pkg)
			assert.Equal(t, tt.expected, actual)
			tt.isError(t, err)
		})
	}
}

func TestIsPackageIgnored(t *testing.T) {
	tests := []struct {
		desc     string
		pkg      string
		mockOS   *mockOS
		expected bool
	}{
		{
			desc: "Test Getwd returns error, returns false",
			pkg:  "",
			mockOS: func() *mockOS {
				m := new(mockOS)

				m.On("Getwd").Return("", errors.New("error getting current dir"))
				return m
			}(),
			expected: false,
		},
		{
			desc: "Test package is not ignored, returns false",
			pkg:  "/a/test/root/package/notignored",
			mockOS: func() *mockOS {
				m := new(mockOS)

				m.On("Getwd").Return("/a/test/root/package", nil)
				return m
			}(),
			expected: false,
		},
		{
			desc: "Test package is ignored, returns true",
			pkg:  "/a/test/root/package/handlers",
			mockOS: func() *mockOS {
				m := new(mockOS)

				m.On("Getwd").Return("/a/test/root/package", nil)
				return m
			}(),
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			defaultOS = tt.mockOS
			actual := isPackageIgnored(tt.pkg)
			assert.Equal(t, tt.expected, actual)
		})
	}
}
