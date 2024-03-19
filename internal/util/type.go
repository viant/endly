package util

import "strings"

// SimpleTypeName returns simple type name
func SimpleTypeName(typeName string) string {
	if index := strings.Index(typeName, "."); index != -1 {
		return string(typeName[index+1:])
	}
	return typeName
}
