package main

import (
	"errors"
	"go/ast"
	"go/parser"
	"go/token"
	path "path/filepath"
	"strings"

	"github.com/astaxie/beego/swagger"
)

const (
	basePath = "/v1"

	chiPath          = "pkg/router/routes.go"
	handlerPrefixCHI = "github.com/zalora/doraemon/handlers"
)

var (
	httpMethods = []string{
		"Get", "Head", "Post", "Put", "Patch",
		"Delete", "Connect", "Options", "Trace",
	}
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

	definedRoutes := chiDefinedRoutes(f)
	for rt, item := range chiAPIs {
		if !isValidRoute(rt, definedRoutes) {
			ColorLog("[WARN] %s is not a valid route\n", rt)
			continue
		}

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

	rootapi.Tags = append(rootapi.Tags, generateChiTags(f, fset)...)

	return nil
}

func isValidRoute(route string, definedRoutes map[string]struct{}) bool {
	route = path.Join(basePath, route)
	_, ok := definedRoutes[route]
	return ok
}

func appendTag(op *swagger.Operation, tag string) {
	if op == nil {
		return
	}

	op.Tags = append(op.Tags, tag)
}

func chiDefinedRoutes(node *ast.File) map[string]struct{} {
	for _, decl := range node.Decls {
		funcDecl, ok := decl.(*ast.FuncDecl)
		if !ok {
			continue
		}

		if funcDecl.Name.Name != "New" {
			continue
		}

		funcLit, err := findMuxGroupArg(funcDecl)
		if err != nil {
			continue
		}

		return traverseRoutes(funcLit, "")
	}

	return nil
}

func generateChiTags(node *ast.File, fset *token.FileSet) []swagger.Tag {
	var lineRouteMap map[int]string
	for _, decl := range node.Decls {
		funcDecl, ok := decl.(*ast.FuncDecl)
		if !ok {
			continue
		}

		if funcDecl.Name.Name != "New" {
			continue
		}

		funcLit, err := findMuxGroupArg(funcDecl)
		if err != nil {
			continue
		}

		route, err := findRouteCall(funcLit, fset)
		if err != nil {
			continue
		}

		lineRouteMap = extractLineRouteMap(route, fset)
		break
	}

	lineCommentMap := extractLineCommentMap(node.Comments, fset)

	var tags []swagger.Tag
	for line, route := range lineRouteMap {
		tags = append(tags, swagger.Tag{
			Name:        route,
			Description: lineCommentMap[line-1],
		})
	}

	return tags
}

func assertCallExpression(callExpr *ast.CallExpr, object string, functions []string) bool {
	selExpr, ok := callExpr.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}

	ident, ok := selExpr.X.(*ast.Ident)
	if !ok {
		return false
	}

	if ident.Obj == nil {
		return false
	}

	if ident.Obj.Name != object {
		return false
	}

	for _, function := range functions {
		if selExpr.Sel.Name == function {
			return true
		}
	}

	return false
}

func extractLineRouteMap(callExpr *ast.CallExpr, fset *token.FileSet) map[int]string {
	if len(callExpr.Args) != 2 {
		return nil
	}

	fnExpr := callExpr.Args[1]

	fn, ok := fnExpr.(*ast.FuncLit)
	if !ok {
		return nil
	}

	lineRouteMap := make(map[int]string)
	for _, stmt := range fn.Body.List {
		exprStmt, ok := stmt.(*ast.ExprStmt)
		if !ok {
			continue
		}

		callExpr, ok := exprStmt.X.(*ast.CallExpr)
		if !ok {
			continue
		}

		if len(callExpr.Args) != 2 {
			return nil
		}

		patternExpr := callExpr.Args[0]

		patternBasicLit, ok := patternExpr.(*ast.BasicLit)
		if !ok {
			return nil
		}

		pattern := strings.Trim(patternBasicLit.Value, `"/`)

		selectorExpr, ok := callExpr.Fun.(*ast.SelectorExpr)
		if !ok {
			continue
		}

		if selectorExpr.Sel.Name != "Route" {
			continue
		}

		pos := fset.Position(selectorExpr.Pos())
		lineRouteMap[pos.Line] = pattern
	}

	return lineRouteMap
}

func findMuxGroupArg(funcDecl *ast.FuncDecl) (*ast.FuncLit, error) {
	for _, stmt := range funcDecl.Body.List {
		expr, ok := stmt.(*ast.ExprStmt)
		if !ok {
			continue
		}

		callExpr, ok := expr.X.(*ast.CallExpr)
		if !ok {
			continue
		}

		// mux.Group(func(r chi.Router) { ... }
		if !assertCallExpression(callExpr, "mux", []string{"Group"}) {
			continue
		}

		if len(callExpr.Args) != 1 {
			continue
		}

		funcLit, ok := callExpr.Args[0].(*ast.FuncLit)
		if !ok {
			continue
		}

		return funcLit, nil
	}

	return nil, errors.New("mux group arg is not found")
}

func traverseRoutes(funcLit *ast.FuncLit, route string) map[string]struct{} {
	routes := make(map[string]struct{})
	for _, stmt := range funcLit.Body.List {
		expr, ok := stmt.(*ast.ExprStmt)
		if !ok {
			continue
		}

		callExpr, ok := expr.X.(*ast.CallExpr)
		if !ok {
			continue
		}

		if assertCallExpression(callExpr, "r", []string{"Route"}) {
			localRoute := route + strings.Trim(callExpr.Args[0].(*ast.BasicLit).Value, `"`)
			localRoute = strings.TrimRight(localRoute, `/`)
			localRoute = toSwaggerPathKey(localRoute)
			subRoutes := traverseRoutes(callExpr.Args[1].(*ast.FuncLit), localRoute)
			for k, v := range subRoutes {
				routes[k] = v
			}
			continue
		}

		if assertCallExpression(callExpr, "r", httpMethods) {
			localRoute := route + "/" + strings.Trim(callExpr.Args[0].(*ast.BasicLit).Value, `"/`)
			localRoute = strings.TrimRight(localRoute, `/`)
			localRoute = toSwaggerPathKey(localRoute)
			routes[localRoute] = struct{}{}
			continue
		}
	}

	return routes
}

func toSwaggerPathKey(path string) string {
	path = strings.ReplaceAll(path, "{", ":")
	path = strings.ReplaceAll(path, "}", "")
	return path
}

func findRouteCall(funcLit *ast.FuncLit, fset *token.FileSet) (*ast.CallExpr, error) {
	for _, stmt := range funcLit.Body.List {
		expr, ok := stmt.(*ast.ExprStmt)
		if !ok {
			continue
		}

		callExpr, ok := expr.X.(*ast.CallExpr)
		if !ok {
			continue
		}

		// r.Route("v1", func(r chi.Router) { ... })
		if !assertCallExpression(callExpr, "r", []string{"Route"}) {
			continue
		}

		return callExpr, nil
	}

	return nil, errors.New("no route is found")
}

func extractLineCommentMap(comments []*ast.CommentGroup, fset *token.FileSet) map[int]string {
	lineCommentMap := make(map[int]string)
	for _, cg := range comments {
		for _, c := range cg.List {
			lineCommentMap[fset.Position(c.Pos()).Line] = strings.TrimLeft(c.Text, "/ ")
		}
	}

	return lineCommentMap
}

func isCHI(pkgpath string) bool {
	return strings.HasPrefix(pkgpath, handlerPrefixCHI)
}
