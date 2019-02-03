package main

import (
	"fmt"
	"github.com/viant/endly/util/adapter"
	"github.com/viant/toolbox"
	"io/ioutil"
	_ "k8s.io/client-go/kubernetes/typed/apps/v1"
	_ "k8s.io/client-go/kubernetes/typed/core/v1"
	_ "k8s.io/client-go/kubernetes/typed/networking/v1"
	_ "k8s.io/client-go/kubernetes/typed/storage/v1"

	"log"
	"path"
	"strings"
)

func main() {
	currentPath := toolbox.CallerDirectory(3)
	parent, _ := path.Split(string(currentPath[:len(currentPath)-1]))
	goPath := string(parent[:strings.Index(parent, "/src/")])

	generateCode(goPath, parent, "core/v1")
	generateCode(goPath, parent, "apps/v1")
	generateCode(goPath, parent, "storage/v1")
	generateCode(goPath, parent, "networking/v1")

}

func generateCode(goPath string, parent, suffix string) {
	corePath := path.Join(goPath, fmt.Sprintf("src/k8s.io/client-go/kubernetes/typed/%v/", suffix))
	gen := adapter.New()
	generated, err := gen.GenerateMatched(corePath, func(typeName string) bool {
		return strings.HasSuffix(typeName, "Interface")
	}, func(receiver *toolbox.FunctionInfo) bool {
		return len(receiver.ParameterFields) > 0

	}, func(typeName string, receiver *toolbox.FunctionInfo) string {
		prefix := strings.Replace(typeName, "Interface", "", 1)
		return prefix + receiver.Name + "Request"
	})
	if err != nil {
		log.Fatal(err)
	}
	for k, v := range generated {
		name := getFilename(k)
		filename := path.Join(parent, suffix, name)
		code := "package v1\n\n" + v
		if err := ioutil.WriteFile(filename, []byte(code), 0644); err != nil {
			log.Fatal(err)
		}
	}
}

func getFilename(name string) string {
	name = strings.Replace(name, "Interface", "", 1)
	return toolbox.ToCaseFormat(name+"Contract", toolbox.CaseUpperCamel, toolbox.CaseLowerUnderscore) + ".go"
}
