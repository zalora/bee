package main

import "strings"

const (
	handlerPrefixCHI = "github.com/zalora/doraemon/handlers"
)

func isCHI(pkgpath string) bool {
	return strings.HasPrefix(pkgpath, handlerPrefixCHI)
}
