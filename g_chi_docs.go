package main

import "strings"

func isCHI(controllerName string) bool {
	if controllerName == "" {
		return true
	}

	if !strings.HasSuffix(controllerName, "Controller") {
		return true
	}

	return false
}
