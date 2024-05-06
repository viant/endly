package main

import (
	"fmt"
	"github.com/viant/endly/internal/util/adapter"
	"github.com/viant/toolbox"
	"io/ioutil"
	"log"
	"path"
	//"github.com/docker/docker/client"
	"strings"
)

func main() {
	currentPath := toolbox.CallerDirectory(3)
	parent, _ := path.Split(string(currentPath[:len(currentPath)-1]))
	goPath := string(parent[:strings.Index(parent, "/src/")])
	err := generateCode(goPath, parent)
	if err != nil {
		panic(err)
	}
}

func generateCode(goPath string, parent string) error {
	sourcePath := path.Join(goPath, fmt.Sprintf("src/github.com/docker/docker/client"))

	fmt.Printf("%v\n", sourcePath)
	gen := adapter.New()
	generated, err := gen.GenerateMatched(sourcePath, func(typeName string) bool {
		return typeName == "Client"
	}, func(receiver *toolbox.FunctionInfo) bool {
		if receiver.Name != strings.Title(receiver.Name) {
			return false
		}
		if len(receiver.ParameterFields) == 0 {
			return false
		}
		for _, params := range receiver.ParameterFields {
			if params.TypeName == "context.Context" {
				return true
			}
		}
		return false
	}, func(meta *adapter.TypeMeta, receiver *toolbox.FunctionInfo) {
		meta.TypeName = receiver.Name + "Request"
		meta.ID = toolbox.ToCaseFormat(meta.TypeName, toolbox.CaseUpperCamel, toolbox.CaseLowerCamel)

		for _, param := range receiver.ParameterFields {
			if strings.Contains(strings.ToLower(param.Name), "option") {
				meta.Embed = true
				param.Tag = "`" + `json:",inline" yaml:",inline"` + "`"
				break
			}
		}

	})

	for _, v := range generated {
		name := "contract_gen.go"
		filename := path.Join(parent, name)
		code := fmt.Sprintf("package %s\n\n", "docker") + v
		fmt.Printf("%v \b %v\n", filename, code)

		if err := ioutil.WriteFile(filename, []byte(code), 0644); err != nil {
			log.Fatal(err)
		}
	}
	return err
}
