package main

import (
	"go/parser"
	"go/token"
	path "path/filepath"
	"strings"

	"github.com/astaxie/beego/swagger"
)

const (
	chiPath          = "pkg/router/routes.go"
	handlerPrefixCHI = "github.com/zalora/doraemon/pkg/api"
)

func generateChiDocs(curpath string) error {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(
		fset,
		path.Join(curpath, chiPath),
		nil,
		parser.ParseComments)
	if err != nil {
		return err
	}

	for _, im := range f.Imports {
		var localName string
		if im.Name != nil {
			localName = im.Name.Name
		}

		analisyscontrollerPkg(localName, im.Path.Value)
	}

	for rt, item := range chiAPIs {
		baseURLSplit := strings.Split(rt, "/")
		if len(baseURLSplit) <= 1 {
			continue
		}
		tag := baseURLSplit[1]

		appendTag(item.Get, tag)
		appendTag(item.Post, tag)
		appendTag(item.Put, tag)
		appendTag(item.Patch, tag)
		appendTag(item.Head, tag)
		appendTag(item.Delete, tag)
		appendTag(item.Options, tag)

		if len(rootapi.Paths) == 0 {
			rootapi.Paths = make(map[string]*swagger.Item)
		}

		rt = urlReplace(rt)
		rootapi.Paths[rt] = item
	}

	return nil
}

func appendTag(op *swagger.Operation, tag string) {
	if op == nil {
		return
	}

	op.Tags = append(op.Tags, tag)
}

func isCHI(pkgpath string) bool {
	return strings.HasPrefix(pkgpath, handlerPrefixCHI)
}
