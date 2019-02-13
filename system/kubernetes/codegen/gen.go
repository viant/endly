package main

import (
	"fmt"
	"github.com/viant/endly/util/adapter"
	"github.com/viant/toolbox"
	"io/ioutil"
	_ "k8s.io/client-go/kubernetes/typed/apps/v1"
	_ "k8s.io/client-go/kubernetes/typed/batch/v1"
	_ "k8s.io/client-go/kubernetes/typed/batch/v1beta1"
	_ "k8s.io/client-go/kubernetes/typed/core/v1"
	_ "k8s.io/client-go/kubernetes/typed/extensions/v1beta1"
	_ "k8s.io/client-go/kubernetes/typed/rbac/v1"

	_ "k8s.io/client-go/kubernetes/typed/autoscaling/v1"
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

	generateCode(goPath, parent, "autoscaling/v1")
	generateCode(goPath, parent, "core/v1")
	generateCode(goPath, parent, "apps/v1")
	generateCode(goPath, parent, "apps/v1beta2")

	generateCode(goPath, parent, "batch/v1")
	generateCode(goPath, parent, "batch/v1beta1")
	generateCode(goPath, parent, "extensions/v1beta1")
	generateCode(goPath, parent, "rbac/v1")
	generateCode(goPath, parent, "policy/v1beta1")

	generateCode(goPath, parent, "storage/v1")
	generateCode(goPath, parent, "networking/v1")

}

func generateCode(goPath string, parent, apiVersion string) {
	corePath := path.Join(goPath, fmt.Sprintf("src/k8s.io/client-go/kubernetes/typed/%v/", apiVersion))
	gen := adapter.New()
	generated, err := gen.GenerateMatched(corePath, func(typeName string) bool {
		return strings.HasSuffix(typeName, "Interface")
	}, func(receiver *toolbox.FunctionInfo) bool {
		return len(receiver.ParameterFields) > 0

	}, func(meta *adapter.TypeMeta, receiver *toolbox.FunctionInfo) {
		typeNamePrefix := strings.Replace(meta.SourceType, "Interface", "", 1)
		meta.TypeName = typeNamePrefix + receiver.Name + "Request"
		API := strings.Replace(apiVersion, "core/", "", 1)
		meta.ID = API + "." + strings.Replace(meta.SimpleOwnerType, "Interface", "", 1) + "." + meta.Func
		if receiver.Name != "Patch" {
			//if receiver.Name == "Get" || receiver.Name == "List" || len(receiver.ParameterFields) == 1 {
			meta.Embed = true
		}
	})

	apiParts := strings.Split(apiVersion, "/")
	packageName := apiParts[len(apiParts)-1]
	if err != nil {
		log.Fatal(err)
	}
	for k, v := range generated {
		name := getFilename(k)
		filename := path.Join(parent, apiVersion, name)
		code := fmt.Sprintf("package %s\n\n", packageName) + v
		if err := ioutil.WriteFile(filename, []byte(code), 0644); err != nil {
			log.Fatal(err)
		}
	}
}

func getFilename(name string) string {
	name = strings.Replace(name, "Interface", "", 1)
	return toolbox.ToCaseFormat(name+"Contract", toolbox.CaseUpperCamel, toolbox.CaseLowerUnderscore) + ".go"
}
