// Copyright 2013 bee authors
//
// Licensed under the Apache License, Version 2.0 (the "License"): you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS, WITHOUT
// WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the
// License for the specific language governing permissions and limitations
// under the License.

package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"unicode"

	"gopkg.in/yaml.v2"

	"github.com/astaxie/beego/swagger"
	"github.com/astaxie/beego/utils"
)

const (
	ajson  = "application/json"
	axml   = "application/xml"
	aplain = "text/plain"
	ahtml  = "text/html"
	aform  = "multipart/form-data"

	content_type_thrift_binary_webcontent_v1 = "application/vnd.zalora.webcontent.v1+thrift.binary"
	content_type_thrift_json_webcontent_v1   = "application/vnd.zalora.webcontent.v1+thrift.json"
	content_type_thrift_binary               = "application/vnd.apache.thrift.binary"
	content_type_thrift_json                 = "application/vnd.apache.thrift.json"
)

var pkgCache map[string]struct{} //pkg:controller:function:comments comments: key:value
var controllerComments map[string]string
var importlist map[string]string
var controllerList map[string]map[string]*swagger.Item //controllername Paths items
var modelsList map[string]map[string]swagger.Schema
var rootapi swagger.Swagger

func init() {
	pkgCache = make(map[string]struct{})
	controllerComments = make(map[string]string)
	importlist = make(map[string]string)
	controllerList = make(map[string]map[string]*swagger.Item)
	modelsList = make(map[string]map[string]swagger.Schema)
}

func generateDocs(curpath string) {
	fset := token.NewFileSet()

	f, err := parser.ParseFile(fset, path.Join(curpath, "routers", "router.go"), nil, parser.ParseComments)

	if err != nil {
		ColorLog("[ERRO] parse router.go error\n")
		os.Exit(2)
	}

	rootapi.Infos = swagger.Information{}
	rootapi.SwaggerVersion = "2.0"
	//analysis API comments
	if f.Comments != nil {
		for _, c := range f.Comments {
			for _, s := range strings.Split(c.Text(), "\n") {
				if strings.HasPrefix(s, "@APIVersion") {
					rootapi.Infos.Version = strings.TrimSpace(s[len("@APIVersion"):])
				} else if strings.HasPrefix(s, "@Title") {
					rootapi.Infos.Title = strings.TrimSpace(s[len("@Title"):])
				} else if strings.HasPrefix(s, "@Description") {
					rootapi.Infos.Description = strings.TrimSpace(s[len("@Description"):])
				} else if strings.HasPrefix(s, "@TermsOfServiceUrl") {
					rootapi.Infos.TermsOfService = strings.TrimSpace(s[len("@TermsOfServiceUrl"):])
				} else if strings.HasPrefix(s, "@Contact") {
					rootapi.Infos.Contact.EMail = strings.TrimSpace(s[len("@Contact"):])
				} else if strings.HasPrefix(s, "@Name") {
					rootapi.Infos.Contact.Name = strings.TrimSpace(s[len("@Name"):])
				} else if strings.HasPrefix(s, "@URL") {
					rootapi.Infos.Contact.URL = strings.TrimSpace(s[len("@URL"):])
				} else if strings.HasPrefix(s, "@License") {
					if rootapi.Infos.License == nil {
						rootapi.Infos.License = &swagger.License{Name: strings.TrimSpace(s[len("@License"):])}
					} else {
						rootapi.Infos.License.Name = strings.TrimSpace(s[len("@License"):])
					}
				} else if strings.HasPrefix(s, "@LicenseUrl") {
					if rootapi.Infos.License == nil {
						rootapi.Infos.License = &swagger.License{URL: strings.TrimSpace(s[len("@LicenseUrl"):])}
					} else {
						rootapi.Infos.License.URL = strings.TrimSpace(s[len("@LicenseUrl"):])
					}
				} else if strings.HasPrefix(s, "@Schemes") {
					rootapi.Schemes = strings.Split(strings.TrimSpace(s[len("@Schemes"):]), ",")
				} else if strings.HasPrefix(s, "@Host") {
					rootapi.Host = strings.TrimSpace(s[len("@Host"):])
				}
			}
		}
	}
	// analisys controller package
	for _, im := range f.Imports {
		localName := ""
		if im.Name != nil {
			localName = im.Name.Name
		}
		analisyscontrollerPkg(localName, im.Path.Value)
	}
	for _, d := range f.Decls {
		switch specDecl := d.(type) {
		case *ast.FuncDecl:
			for _, l := range specDecl.Body.List {
				switch stmt := l.(type) {
				case *ast.AssignStmt:
					for _, l := range stmt.Rhs {
						if v, ok := l.(*ast.CallExpr); ok {
							// analisys NewNamespace, it will return version and the subfunction
							if selName := v.Fun.(*ast.SelectorExpr).Sel.String(); selName != "NewNamespace" {
								continue
							}
							version, params := analisysNewNamespace(v)
							if rootapi.BasePath == "" && version != "" {
								rootapi.BasePath = version
							}
							for _, p := range params {
								switch pp := p.(type) {
								case *ast.CallExpr:
									controllerName := ""
									if selname := pp.Fun.(*ast.SelectorExpr).Sel.String(); selname == "NSNamespace" {
										s, params := analisysNewNamespace(pp)
										for _, sp := range params {
											switch pp := sp.(type) {
											case *ast.CallExpr:
												if pp.Fun.(*ast.SelectorExpr).Sel.String() == "NSInclude" {
													controllerName = analisysNSInclude(s, pp)
													if v, ok := controllerComments[controllerName]; ok {
														rootapi.Tags = append(rootapi.Tags, swagger.Tag{
															Name:        strings.Trim(s, "/"),
															Description: v,
														})
													}
												}
											}
										}
									} else if selname == "NSInclude" {
										controllerName = analisysNSInclude("", pp)
										if v, ok := controllerComments[controllerName]; ok {
											rootapi.Tags = append(rootapi.Tags, swagger.Tag{
												Name:        controllerName, // if the NSInclude has no prefix, we use the controllername as the tag
												Description: v,
											})
										}
									}
								}
							}
						}

					}
				}
			}
		}
	}
	os.Mkdir(path.Join(curpath, "swagger"), 0755)
	fd, err := os.Create(path.Join(curpath, "swagger", "swagger.json"))
	fdyml, err := os.Create(path.Join(curpath, "swagger", "swagger.yml"))
	if err != nil {
		panic(err)
	}
	defer fdyml.Close()
	defer fd.Close()
	dt, err := json.MarshalIndent(rootapi, "", "    ")
	dtyml, erryml := yaml.Marshal(rootapi)
	if err != nil || erryml != nil {
		panic(err)
	}
	_, err = fd.Write(dt)
	_, erryml = fdyml.Write(dtyml)
	if err != nil || erryml != nil {
		panic(err)
	}
}

// return version and the others params
func analisysNewNamespace(ce *ast.CallExpr) (first string, others []ast.Expr) {
	for i, p := range ce.Args {
		if i == 0 {
			switch pp := p.(type) {
			case *ast.BasicLit:
				first = strings.Trim(pp.Value, `"`)
			}
			continue
		}
		others = append(others, p)
	}
	return
}

func analisysNSInclude(baseurl string, ce *ast.CallExpr) string {
	cname := ""
	for _, p := range ce.Args {
		x := p.(*ast.UnaryExpr).X.(*ast.CompositeLit).Type.(*ast.SelectorExpr)
		if v, ok := importlist[fmt.Sprint(x.X)]; ok {
			cname = v + x.Sel.Name
		}
		if apis, ok := controllerList[cname]; ok {
			for rt, item := range apis {
				tag := ""
				if baseurl != "" {
					rt = baseurl + rt
					tag = strings.Trim(baseurl, "/")
				} else {
					tag = cname
				}
				if item.Get != nil {
					item.Get.Tags = append(item.Get.Tags, tag)
				}
				if item.Post != nil {
					item.Post.Tags = append(item.Post.Tags, tag)
				}
				if item.Put != nil {
					item.Put.Tags = append(item.Put.Tags, tag)
				}
				if item.Patch != nil {
					item.Patch.Tags = append(item.Patch.Tags, tag)
				}
				if item.Head != nil {
					item.Head.Tags = append(item.Head.Tags, tag)
				}
				if item.Delete != nil {
					item.Delete.Tags = append(item.Delete.Tags, tag)
				}
				if item.Options != nil {
					item.Options.Tags = append(item.Options.Tags, tag)
				}
				if len(rootapi.Paths) == 0 {
					rootapi.Paths = make(map[string]*swagger.Item)
				}
				rt = urlReplace(rt)
				rootapi.Paths[rt] = item
			}
		}
	}
	return cname
}

func analisyscontrollerPkg(localName, pkgpath string) {
	pkgpath = strings.Trim(pkgpath, "\"")
	if isSystemPackage(pkgpath) {
		return
	}
	if pkgpath == "github.com/astaxie/beego" {
		return
	}
	if localName != "" {
		importlist[localName] = pkgpath
	} else {
		pps := strings.Split(pkgpath, "/")
		importlist[pps[len(pps)-1]] = pkgpath
	}

	// Lets search for the beginning of package path in the current working
	// directory (cwd). If found, replace the beginning of the package path
	// in current working directory with the rest of the package path.
	//
	// cwd: /Users/foo/my/path/to/the/<project>
	// pkgpath: github.com/<user>/<project>/pkg/server/handlers
	//
	// will become: /Users/foo/my/path/to/the/<project>/pkg/server/handlers

	wd, err := os.Getwd()
	if err != nil {
		panic(fmt.Sprintf("could not get current working directory: %v", err))
	}

	pkgPathParts := strings.Split(pkgpath, string(os.PathSeparator))
	if len(pkgPathParts) == 0 {
		panic(fmt.Sprintf("invalid package path: %s", pkgpath))
	}

	// Extract the project name from the package path.
	// Assumption: package path has always the form github.com/<user>/<project>
	project := pkgPathParts[2]

	if !strings.Contains(wd, project) {
		// If we dont find the project in the cwd, lets not generate docs for it.
		return
	}

	idx := strings.Index(pkgpath, project)
	if idx < 0 {
		panic(fmt.Sprintf("package path does not contain the project %q: %s",
			project, pkgpath))
	}

	// github.com/<user>/<project>/modules/foobar -> /modules/foobar
	offset := idx + len(project)
	fp := filepath.Join(wd, pkgpath[offset:])

	pkgRealpath, _ := filepath.EvalSymlinks(fp)

	if pkgRealpath != "" {
		if _, ok := pkgCache[pkgpath]; ok {
			return
		}
		pkgCache[pkgpath] = struct{}{}
	} else {
		ColorLog("[ERRO] the %s pkg not exist in gopath\n", pkgpath)
		os.Exit(1)
	}
	fileSet := token.NewFileSet()
	astPkgs, err := parser.ParseDir(fileSet, pkgRealpath, func(info os.FileInfo) bool {
		name := info.Name()
		return !info.IsDir() && !strings.HasPrefix(name, ".") && strings.HasSuffix(name, ".go")
	}, parser.ParseComments)

	if err != nil {
		ColorLog("[ERRO] the %s pkg parser.ParseDir error\n", pkgpath)
		os.Exit(1)
	}
	for _, pkg := range astPkgs {
		for _, fl := range pkg.Files {
			for _, d := range fl.Decls {
				switch specDecl := d.(type) {
				case *ast.FuncDecl:
					if specDecl.Recv != nil && len(specDecl.Recv.List) > 0 {
						if t, ok := specDecl.Recv.List[0].Type.(*ast.StarExpr); ok {
							// parse controller method
							parserComments(specDecl.Doc, specDecl.Name.String(), fmt.Sprint(t.X), pkgpath)
						}
					}
				case *ast.GenDecl:
					if specDecl.Tok == token.TYPE {
						for _, s := range specDecl.Specs {
							switch tp := s.(*ast.TypeSpec).Type.(type) {
							case *ast.StructType:
								_ = tp.Struct
								//parse controller definition comments
								if strings.TrimSpace(specDecl.Doc.Text()) != "" {
									controllerComments[pkgpath+s.(*ast.TypeSpec).Name.String()] = specDecl.Doc.Text()
								}
							}
						}
					}
				}
			}
		}
	}
}

func isSystemPackage(pkgpath string) bool {
	goroot := runtime.GOROOT()
	if goroot == "" {
		panic("goroot is empty, do you install Go right?")
	}
	wg, _ := filepath.EvalSymlinks(filepath.Join(goroot, "src", "pkg", pkgpath))
	if utils.FileExists(wg) {
		return true
	}

	//TODO(zh):support go1.4
	wg, _ = filepath.EvalSymlinks(filepath.Join(goroot, "src", pkgpath))
	if utils.FileExists(wg) {
		return true
	}

	return false
}

func peekNextSplitString(ss string) (s string, spacePos int) {
	spacePos = strings.IndexFunc(ss, unicode.IsSpace)
	if spacePos < 0 {
		s = ss
		spacePos = len(ss)
	} else {
		s = strings.TrimSpace(ss[:spacePos])
	}
	return
}

// parse the func comments
func parserComments(comments *ast.CommentGroup, funcName, controllerName, pkgpath string) error {
	var routerPath string
	var HTTPMethod string
	opts := swagger.Operation{
		Responses: make(map[string]swagger.Response),
	}
	if comments != nil && comments.List != nil {
		for _, c := range comments.List {
			t := strings.TrimSpace(strings.TrimLeft(c.Text, "//"))
			if strings.HasPrefix(t, "@router") {
				elements := strings.TrimSpace(t[len("@router"):])
				e1 := strings.SplitN(elements, " ", 2)
				if len(e1) < 1 {
					return errors.New("you should has router infomation")
				}
				routerPath = e1[0]
				if len(e1) == 2 && e1[1] != "" {
					e1 = strings.SplitN(e1[1], " ", 2)
					HTTPMethod = strings.ToUpper(strings.Trim(e1[0], "[]"))
				} else {
					HTTPMethod = "GET"
				}
			} else if strings.HasPrefix(t, "@Title") {
				opts.OperationID = controllerName + "." + strings.TrimSpace(t[len("@Title"):])
			} else if strings.HasPrefix(t, "@Description") {
				opts.Description = strings.TrimSpace(t[len("@Description"):])
			} else if strings.HasPrefix(t, "@Summary") {
				opts.Summary = strings.TrimSpace(t[len("@Summary"):])
			} else if strings.HasPrefix(t, "@Success") {
				ss := strings.TrimSpace(t[len("@Success"):])
				rs := swagger.Response{}
				respCode, pos := peekNextSplitString(ss)
				ss = strings.TrimSpace(ss[pos:])
				respType, pos := peekNextSplitString(ss)
				if respType == "{object}" || respType == "{array}" {
					isArray := respType == "{array}"
					ss = strings.TrimSpace(ss[pos:])
					schemaName, pos := peekNextSplitString(ss)
					if schemaName == "" {
						ColorLog("[ERRO][%s.%s] Schema must follow {object} or {array}\n", controllerName, funcName)
						os.Exit(-1)
					}
					if strings.HasPrefix(schemaName, "[]") {
						schemaName = schemaName[2:]
						isArray = true
					}
					schema := swagger.Schema{}
					if sType, ok := basicTypes[schemaName]; ok {
						typeFormat := strings.Split(sType, ":")
						schema.Type = typeFormat[0]
						schema.Format = typeFormat[1]
					} else {
						cmpath, m, mod, realTypes := getModel(schemaName)
						schema.Ref = "#/definitions/" + m
						if _, ok := modelsList[pkgpath+controllerName]; !ok {
							modelsList[pkgpath+controllerName] = make(map[string]swagger.Schema, 0)
						}
						modelsList[pkgpath+controllerName][schemaName] = mod
						appendModels(cmpath, pkgpath, controllerName, realTypes)
					}
					if isArray {
						rs.Schema = &swagger.Schema{
							Type:  "array",
							Items: &schema,
						}
					} else {
						rs.Schema = &schema
					}
					rs.Description = strings.TrimSpace(ss[pos:])
				} else {
					rs.Description = strings.TrimSpace(ss)
				}
				opts.Responses[respCode] = rs
			} else if strings.HasPrefix(t, "@Param") {
				para := swagger.Parameter{}
				p := getparams(strings.TrimSpace(t[len("@Param "):]))
				if len(p) < 4 {
					panic(controllerName + "_" + funcName + "'s comments @Param at least should has 4 params")
				}
				para.Name = p[0]
				switch p[1] {
				case "query":
					fallthrough
				case "header":
					fallthrough
				case "path":
					fallthrough
				case "formData":
					fallthrough
				case "body":
					break
				default:
					ColorLog("[WARN][%s.%s] Unknow param location: %s, Possible values are `query`, `header`, `path`, `formData` or `body`.\n", controllerName, funcName, p[1])
				}
				para.In = p[1]
				pp := strings.Split(p[2], ".")
				typ := pp[len(pp)-1]
				if len(pp) >= 2 {
					cmpath, m, mod, realTypes := getModel(p[2])
					para.Schema = &swagger.Schema{
						Ref: "#/definitions/" + m,
					}
					if _, ok := modelsList[pkgpath+controllerName]; !ok {
						modelsList[pkgpath+controllerName] = make(map[string]swagger.Schema, 0)
					}
					modelsList[pkgpath+controllerName][typ] = mod
					appendModels(cmpath, pkgpath, controllerName, realTypes)
				} else {
					isArray := false
					paraType := ""
					paraFormat := ""
					if strings.HasPrefix(typ, "[]") {
						typ = typ[2:]
						isArray = true
					}

					if typ == "string" || typ == "number" || typ == "integer" || typ == "boolean" ||
						typ == "array" || typ == "file" {
						paraType = typ
					} else if sType, ok := basicTypes[typ]; ok {
						typeFormat := strings.Split(sType, ":")
						paraType = typeFormat[0]
						paraFormat = typeFormat[1]
					} else if typ == "enum" {
						paraType = "string"
						para.Enum = strings.Split(p[4], ",")
						if len(p) > 6 {
							para.Default = p[5]
						}
					} else {
						ColorLog("[WARN][%s.%s] Unknow param type: %s\n", controllerName, funcName, typ)
					}
					if isArray {
						para.Type = "array"
						para.Items = &swagger.ParameterItems{
							Type:   paraType,
							Format: paraFormat,
						}
					} else {
						para.Type = paraType
						para.Format = paraFormat
					}
				}
				para.Required, _ = strconv.ParseBool(p[3])
				para.Description = strings.Trim(p[len(p)-1], `" `)
				opts.Parameters = append(opts.Parameters, para)
			} else if strings.HasPrefix(t, "@Failure") {
				rs := swagger.Response{}
				st := strings.TrimSpace(t[len("@Failure"):])
				var cd []rune
				var start bool
				for i, s := range st {
					if unicode.IsSpace(s) {
						if start {
							rs.Description = strings.TrimSpace(st[i+1:])
							break
						} else {
							continue
						}
					}
					start = true
					cd = append(cd, s)
				}
				opts.Responses[string(cd)] = rs
			} else if strings.HasPrefix(t, "@Deprecated") {
				opts.Deprecated, _ = strconv.ParseBool(strings.TrimSpace(t[len("@Deprecated"):]))
			} else if strings.HasPrefix(t, "@Accept") {
				accepts := strings.Split(strings.TrimSpace(strings.TrimSpace(t[len("@Accept"):])), ",")
				for _, a := range accepts {
					switch a {
					case "json":
						opts.Consumes = append(opts.Consumes, ajson)
						opts.Produces = append(opts.Produces, ajson)
					case "xml":
						opts.Consumes = append(opts.Consumes, axml)
						opts.Produces = append(opts.Produces, axml)
					case "plain":
						opts.Consumes = append(opts.Consumes, aplain)
						opts.Produces = append(opts.Produces, aplain)
					case "html":
						opts.Consumes = append(opts.Consumes, ahtml)
						opts.Produces = append(opts.Produces, ahtml)
					case "thrift_binary":
						opts.Consumes = append(opts.Consumes, content_type_thrift_binary)
						opts.Produces = append(opts.Produces, content_type_thrift_binary)
					case "thrift_json":
						opts.Consumes = append(opts.Consumes, content_type_thrift_json)
						opts.Produces = append(opts.Produces, content_type_thrift_json)
					case "thrift_webcontent_binary":
						opts.Consumes = append(opts.Consumes, content_type_thrift_binary_webcontent_v1)
						opts.Produces = append(opts.Produces, content_type_thrift_binary_webcontent_v1)
					case "thrift_webcontent_json":
						opts.Consumes = append(opts.Consumes, content_type_thrift_json_webcontent_v1)
						opts.Produces = append(opts.Produces, content_type_thrift_json_webcontent_v1)
					case "form":
						opts.Consumes = append(opts.Consumes, aform)
					}
				}
			}
		}
	}
	if routerPath != "" {
		var item *swagger.Item
		if itemList, ok := controllerList[pkgpath+controllerName]; ok {
			if it, ok := itemList[routerPath]; !ok {
				item = &swagger.Item{}
			} else {
				item = it
			}
		} else {
			controllerList[pkgpath+controllerName] = make(map[string]*swagger.Item)
			item = &swagger.Item{}
		}
		switch HTTPMethod {
		case "GET":
			item.Get = &opts
		case "POST":
			item.Post = &opts
		case "PUT":
			item.Put = &opts
		case "PATCH":
			item.Patch = &opts
		case "DELETE":
			item.Delete = &opts
		case "HEAD":
			item.Head = &opts
		case "OPTIONS":
			item.Options = &opts
		}
		controllerList[pkgpath+controllerName][routerPath] = item
	}
	return nil
}

// analisys params return []string
// @Param	query		form	 string	true		"The email for login"
// [query form string true "The email for login"]
func getparams(str string) []string {
	var s []rune
	var j int
	var start bool
	var r []string
	for i, c := range []rune(str) {
		if len([]rune(str))-1 == i && start {
			s = append(s, c)
			r = append(r, string(s))
		}
		if unicode.IsSpace(c) {
			if !start {
				continue
			} else {
				start = false
				j++
				r = append(r, string(s))
				s = make([]rune, 0)
				continue
			}
		}
		if c == '"' {
			r = append(r, strings.TrimSpace((str[i:])))
			break
		}
		start = true
		s = append(s, c)
	}
	return r
}

func getModel(str string) (pkgpath, objectname string, m swagger.Schema, realTypes []string) {
	strs := strings.Split(str, ".")
	objectname = strs[len(strs)-1]
	pkgpath = strings.Join(strs[:len(strs)-1], "/")
	curpath, _ := os.Getwd()
	pkgRealpath := path.Join(curpath, pkgpath)
	fileSet := token.NewFileSet()
	astPkgs, err := parser.ParseDir(fileSet, pkgRealpath, func(info os.FileInfo) bool {
		name := info.Name()
		return !info.IsDir() && !strings.HasPrefix(name, ".") && strings.HasSuffix(name, ".go")
	}, parser.ParseComments)

	if err != nil {
		ColorLog("[ERRO] the model %s parser.ParseDir error\n", str)
		os.Exit(1)
	}

	var packageName string
	m.Type = "object"
	for _, pkg := range astPkgs {
		for _, fl := range pkg.Files {
			pathInfo, err := generatePathInfo(fl)
			if err != nil {
				ColorLog("[ERRO] failed generating path info: %v", err)
				os.Exit(1)
			}

			for k, d := range fl.Scope.Objects {
				if d.Kind == ast.Typ {
					if k != objectname {
						continue
					}
					packageName = pkg.Name
					pathInfo[packageName] = pkgpath
					parseObject(
						d,
						k,
						&m,
						&realTypes,
						astPkgs,
						packageName,
						pathInfo,
					)
				}
			}
		}
	}
	if m.Title == "" {
		ColorLog("[WARN]can't find the object: %s\n", str)
		// TODO remove when all type have been supported
		//os.Exit(1)
	}
	if len(rootapi.Definitions) == 0 {
		rootapi.Definitions = make(map[string]swagger.Schema)
	}
	objectname = packageName + "." + objectname
	rootapi.Definitions[objectname] = m
	return
}

func parseObject(d *ast.Object, k string, m *swagger.Schema, realTypes *[]string, astPkgs map[string]*ast.Package, packageName string, pathInfo map[string]string) {
	ts, ok := d.Decl.(*ast.TypeSpec)
	if !ok {
		ColorLog("Unknown type without TypeSec: %v\n", d)
		os.Exit(1)
	}
	// TODO support other types, such as `ArrayType`, `MapType`, `InterfaceType` etc...
	switch t := ts.Type.(type) {
	case *ast.StructType:
		parseStruct(m, t, k, packageName, realTypes, pathInfo, astPkgs)
	case *ast.Ident, *ast.ArrayType:
		m.Title = k
		m.Properties = make(map[string]swagger.Propertie)
		mp := constructObjectPropertie(t, packageName, realTypes, pathInfo)
		m.Properties[k] = mp
	}

	return
}

func isBasicType(Type string) bool {
	if _, ok := basicTypes[Type]; ok {
		return true
	}
	return false
}

// refer to builtin.go
var basicTypes = map[string]string{
	"bool": "boolean:",
	"uint": "integer:int32", "uint8": "integer:int32", "uint16": "integer:int32", "uint32": "integer:int32", "uint64": "integer:int64",
	"int": "integer:int64", "int8": "integer:int32", "int16:int32": "integer:int32", "int32": "integer:int32", "int64": "integer:int64",
	"uintptr": "integer:int64",
	"float32": "number:float", "float64": "number:double",
	"string":    "string:",
	"complex64": "number:float", "complex128": "number:double",
	"byte": "string:byte", "rune": "string:byte",
}

// regexp get json tag
func grepJSONTag(tag string) string {
	r, _ := regexp.Compile(`json:"([^"]*)"`)
	matches := r.FindAllStringSubmatch(tag, -1)
	if len(matches) > 0 {
		return matches[0][1]
	}
	return ""
}

// append models
func appendModels(cmpath, pkgpath, controllerName string, realTypes []string) {
	var p string
	if cmpath != "" {
		p = strings.Join(strings.Split(cmpath, "/"), ".") + "."
	} else {
		p = ""
	}
	for _, realType := range realTypes {
		if realType != "" && !isBasicType(strings.TrimLeft(realType, "[]")) &&
			!strings.HasPrefix(realType, "map") && !strings.HasPrefix(realType, "&") {
			if _, ok := modelsList[pkgpath+controllerName][p+realType]; ok {
				continue
			}
			_, _, mod, newRealTypes := getModel(realType)
			modelsList[pkgpath+controllerName][realType] = mod
			appendModels(cmpath, pkgpath, controllerName, newRealTypes)
		}
	}
}

func urlReplace(src string) string {
	pt := strings.Split(src, "/")
	for i, p := range pt {
		if len(p) > 0 {
			if p[0] == ':' {
				pt[i] = "{" + p[1:] + "}"
			} else if p[0] == '?' && p[1] == ':' {
				pt[i] = "{" + p[2:] + "}"
			}
		}
	}
	return strings.Join(pt, "/")
}

// constructObjectPropertie construct a swagger.Propertie out of
// an ast.Expr. This function recursively traverse all expression
// until it reaches an object or pre-defined basic golang type.
func constructObjectPropertie(field ast.Expr, packageName string, realTypes *[]string, pathInfo map[string]string) (propertie swagger.Propertie) {

	// basic Go type can be directly translated into swagger type
	// with pre-defined mapping
	if isBasicType(fmt.Sprint(field)) {
		basicType := basicTypes[fmt.Sprint(field)]
		propInfo := strings.Split(basicType, ":")

		if len(propInfo) != 2 {
			ColorLog("[ERRO] basicTypes const is not properly configured for %v", field)
			os.Exit(1)
		}

		propertie.Type = propInfo[0]
		propertie.Format = propInfo[1]

		return
	}

	switch f := field.(type) {
	case *ast.StarExpr:
		object := fmt.Sprint(f.X)
		pkgObject := objectWithPackageName(object, packageName)
		propertie.Ref = "#/definitions/" + pkgObject
		appendObjectToRealTypes(realTypes, pkgObject, pathInfo)
		return
	case *ast.ArrayType:
		object := constructObjectPropertie(
			f.Elt, packageName, realTypes, pathInfo,
		)
		propertie.Type = "array"
		propertie.Items = &object
		return
	case *ast.MapType:
		object := constructObjectPropertie(
			f.Value, packageName, realTypes, pathInfo,
		)
		propertie.Type = "object"
		propertie.AdditionalProperties = &object
		return
	case *ast.StructType:
		object := make(map[string]swagger.Propertie)
		for _, v := range f.Fields.List {
			object[v.Names[0].Name] = constructObjectPropertie(
				v.Type, packageName, realTypes, pathInfo,
			)
		}

		propertie.Type = "object"
		propertie.Properties = object
		return
	// type identity; eg. type anyType int64
	case *ast.Ident:
		return constructObjectPropertie(
			f.Obj.Decl.(*ast.TypeSpec).Type,
			packageName,
			realTypes,
			pathInfo,
		)
	case *ast.SelectorExpr:
		object := fmt.Sprint(f)
		pkgObject := objectWithPackageName(object, packageName)
		propertie.Ref = "#/definitions/" + pkgObject
		appendObjectToRealTypes(realTypes, pkgObject, pathInfo)
		return
	default:
		ColorLog("[WARN] %v type is not handled yet", f)
		return
	}
}

// objectWithPackageName returns an object with package name in format
// packageName.Object. There are two types of object that can be identified
// by ast, the internal and imported objects.
// The imported object comes with `&{PackageName Object}` format and
// the internal object comes with `object` format. In the latter
// case we need to assign the current package name to the object.
func objectWithPackageName(object, packageName string) string {
	if len(strings.Split(object, " ")) == 1 {
		return packageName + "." + object
	}

	object = strings.Replace(object, " ", ".", -1)
	object = strings.Replace(object, "&", "", -1)
	object = strings.Replace(object, "{", "", -1)
	object = strings.Replace(object, "}", "", -1)

	return object
}

// generatePathInfo generates all imported packages in a file into a map.
// the name of the package will be used as the map key, and the path
// to the package will be used as the map value.
func generatePathInfo(file *ast.File) (pathInfo map[string]string, err error) {
	pathInfo = make(map[string]string)
	goSrcPath := os.Getenv("GOPATH") + "/src/"

	// currenPath of where the `bee` is run
	currentPath, err := os.Getwd()
	if err != nil {
		return
	}

	// basePath is the currentPath without GOPATH + src prefix
	// eg. github.com/organization/repository/
	basePath := strings.Replace(currentPath, goSrcPath, "", -1)

	var importPath string
	for _, v := range file.Imports {
		importPath = strings.Trim(v.Path.Value, "\"")
		if !strings.HasPrefix(importPath, basePath) {
			continue
		}
		importPath = strings.Replace(importPath, basePath, "", -1)

		// if the package imported is named, then use the name
		if v.Name != nil {
			pathInfo[v.Name.Name] = importPath
			continue
		}

		packageNames := strings.Split(importPath, "/")
		name := packageNames[len(packageNames)-1]

		pathInfo[name] = importPath
	}

	return
}

// appendObjectToRealTypes append an object with its full path
// from the root package to *realTypes array.
func appendObjectToRealTypes(realTypes *[]string, pkgObject string, pathInfo map[string]string) {
	pkgObjectSplit := strings.Split(pkgObject, ".")

	if len(pkgObjectSplit) != 2 {
		ColorLog("[WARN] %v pkgObject passed to realTypes length should be 2", pkgObjectSplit)
		return
	}

	pkg := pkgObjectSplit[0]
	object := pkgObjectSplit[1]

	realType := pathInfo[pkg] + "." + object
	realType = strings.Replace(realType, "/", ".", -1)
	*realTypes = append(*realTypes, realType)
}

// parseStruct parse a struct type object by iterating over all of the
// fields and create swagger type and definitions out of it
func parseStruct(m *swagger.Schema, structDef *ast.StructType, objectName string, packageName string, realTypes *[]string, pathInfo map[string]string, astPkgs map[string]*ast.Package) {
	m.Title = objectName
	if structDef.Fields.List != nil {
		m.Properties = make(map[string]swagger.Propertie)
		for _, field := range structDef.Fields.List {
			propertie := constructObjectPropertie(
				field.Type, packageName, realTypes, pathInfo,
			)
			if field.Names != nil {

				// set property name as field name
				var name = field.Names[0].Name

				// if no tag skip tag processing
				if field.Tag == nil {
					m.Properties[name] = propertie
					continue
				}

				var tagValues []string
				stag := reflect.StructTag(strings.Trim(field.Tag.Value, "`"))
				tag := stag.Get("json")

				if tag != "" {
					tagValues = strings.Split(tag, ",")
				}

				// dont add property if json tag first value is "-"
				if len(tagValues) == 0 || tagValues[0] != "-" {

					// set property name to the left most json tag value only if is not omitempty
					if len(tagValues) > 0 && tagValues[0] != "omitempty" {
						name = tagValues[0]
					}

					if thrifttag := stag.Get("thrift"); thrifttag != "" {
						ts := strings.Split(thrifttag, ",")
						if ts[0] != "" {
							name = ts[0]
						}
					}
					if required := stag.Get("required"); required != "" {
						m.Required = append(m.Required, name)
					}
					if desc := stag.Get("description"); desc != "" {
						propertie.Description = desc
					}

					m.Properties[name] = propertie
				}
				if ignore := stag.Get("ignore"); ignore != "" {
					continue
				}
			} else {
				for _, pkg := range astPkgs {
					for _, fl := range pkg.Files {
						pathInfo, err := generatePathInfo(fl)
						if err != nil {
							ColorLog("[ERRO] failed generating path info: %v", err)
							os.Exit(1)
						}

						for nameOfObj, obj := range fl.Scope.Objects {
							if obj.Name == fmt.Sprint(field.Type) {
								parseObject(obj, nameOfObj, m, realTypes, astPkgs, pkg.Name, pathInfo)
							}
						}
					}
				}
			}
		}
	}
}
