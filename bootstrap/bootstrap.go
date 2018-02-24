package bootstrap

import (
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	_ "github.com/viant/asc"
	_ "github.com/viant/bgc"
	_ "github.com/viant/endly/static" //load external resource like .csv .json files to mem storage
	_ "github.com/viant/toolbox/storage/aws"
	_ "github.com/viant/toolbox/storage/gs"

	"encoding/json"
	"flag"
	"fmt"
	"github.com/viant/endly"
	"github.com/viant/toolbox"
	"github.com/viant/toolbox/url"
	"gopkg.in/yaml.v2"
	"log"
	"os"
	"path"
	"strings"
	"time"
	"github.com/viant/toolbox/cred"
	"bufio"
	"golang.org/x/crypto/ssh/terminal"
	"syscall"
	"errors"
)

func init() {

	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
	flag.String("r", "run.json", "<path/url to workflow run request in JSON format> ")
	flag.String("w", "manager", "<workflow name>  if both -r and -w valid options are specified, -w is ignored")
	flag.String("t", "*", "<task/s to run>, t='?' to list all tasks for selected workflow")
	flag.String("l", "logs", "<log directory>")
	flag.Bool("d", false, "enable logging")
	flag.Bool("p", false, "print neatly workflow as JSON")
	flag.String("f", "json", "<workflow or request format>, json or yaml")

	flag.Bool("h", false, "print help")
	flag.Bool("v", false, "print version")

	flag.String("s", "", "<serviceID> print service details, -s='*' prints all service IDs")
	flag.String("a", "", "<action> prints action request representation")
	flag.String("i", "", "<coma separated tagID list> to filter")
	flag.String("c", "", "<credential>, generate secret credentials file: ~/.secret/<credential>.json")
	flag.String("k", "", "<private key path>,  works only with -c options, i.e -k=" + path.Join(os.Getenv("HOME"), ".secret/id_rsa.pub"))

}

func Bootstrap() {
	flag.Usage = printHelp
	flag.Parse()
	flagset := make(map[string]string)
	flag.Visit(func(f *flag.Flag) {
		flagset[f.Name] = f.Value.String()
	})

	_, shouldQuit := flagset["v"]
	flagset["v"] = flag.Lookup("v").Value.String()

	if toolbox.AsBoolean(flagset["v"]) {
		printVersion()
		if shouldQuit {
			return
		}
	}

	if _, ok := flagset["h"]; ok {
		printHelp()
		return
	}

	if _, ok := flagset["c"]; ok {
		generateSecret(flag.Lookup("c").Value.String())
		return
	}

	if _, ok := flagset["a"]; ok {
		printServiceActionRequest()
		return
	}
	if _, ok := flagset["s"]; ok {
		printServiceActions()
		return
	}

	request, option, err := getRunRequestWithOptions(flagset)
	if request == nil {
		flagset["r"] = flag.Lookup("r").Value.String()
		flagset["w"] = flag.Lookup("w").Value.String()
		request, option, err = getRunRequestWithOptions(flagset)
		if err != nil && strings.Contains(err.Error(), "failed to locate workflow: manager") {
			printHelp()
			return
		}
	}

	if err != nil {
		log.Fatal(err)
	}
	if value, ok := flagset["p"]; ok && toolbox.AsBoolean(value) {
		printWorkflow(request.WorkflowURL)
		return
	}

	if flagset["t"] == "?" {
		printWorkflowTasks(request.WorkflowURL)
		return
	}

	runner := endly.NewCliRunner()
	err = runner.Run(request, option)
	if err != nil {
		log.Fatal(err)
	}
	time.Sleep(time.Second)
}


func generateSecret(credentialsFile string) {
	secretPath := path.Join(os.Getenv("HOME"), ".secret")
	if !toolbox.FileExists(secretPath) {
		os.Mkdir(secretPath, 0744)
	}
	username, password, err := credentials()
	if err != nil {
		fmt.Printf("\n")
		log.Fatal(err)
	}
	fmt.Println("")
	config := &cred.Config{
		Username: username,
		Password: password,
	}
	var privateKeyPath = flag.Lookup("k").Value.String()
	privateKeyPath = strings.Replace(privateKeyPath, "~", os.Getenv("HOME"), 1)
	if toolbox.FileExists(privateKeyPath) && !cred.IsKeyEncrypted(privateKeyPath) {
		config.PrivateKeyPath = privateKeyPath
	}
	var secretFile = path.Join(secretPath, fmt.Sprintf("%v.json", credentialsFile))
	err = config.Save(secretFile)
	if err != nil {
		fmt.Printf("\n")
		log.Fatal(err)
	}
}

func credentials() (string, string, error) {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Enter Username: ")
	username, _ := reader.ReadString('\n')
	fmt.Print("Enter Password: ")
	bytePassword, err := terminal.ReadPassword(int(syscall.Stdin))
	if err != nil {
		log.Fatal("failed to read password %v", err)
	}
	fmt.Print("\nRetype Password: ")
	bytePassword2, err := terminal.ReadPassword(int(syscall.Stdin))
	if err != nil {
		log.Fatal("failed to read password %v", err)
	}

	password := string(bytePassword)
	if string(bytePassword2) != password {
		return "", "", errors.New("Password did not match")
	}
	return strings.TrimSpace(username), strings.TrimSpace(password), nil
}



func printWorkflowTasks(URL string) {
	workflow, err := getWorkflow(URL)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Fprintf(os.Stderr, "Workflow '%v' (%v) tasks:\n", workflow.Name, workflow.Source.URL)
	for _, task := range workflow.Tasks {
		fmt.Fprintf(os.Stderr, "\t%v: %v\n", task.Name, task.Description)
	}
}

func printServiceActionInfo(renderer *endly.Renderer, info *endly.ActionInfo, color, infoType string, empty interface{}) {
	if info != nil {
		if info.Description != "" {
			renderer.Printf(renderer.ColorText("Description: ", color, "bold")+" %v\n", info.Description)
		}
		if len(info.Examples) > 0 {
			for i, example := range info.Examples {

				renderer.Printf(renderer.ColorText(fmt.Sprintf("Example %v: ", i+1), color, "bold")+" %v %v\n", example.UseCase, infoType)
				aMap, err := toolbox.JSONToMap(example.Data)
				if err == nil {
					buf, _ := json.MarshalIndent(aMap, "", "\t")
					renderer.Println(string(buf))
				} else {
					renderer.Printf("%v\n", example.Data)
				}
			}
		}
	}
	renderer.Printf(renderer.ColorText(fmt.Sprintf("Empty %v: \n", infoType), color, "bold"))
	buf, _ := json.MarshalIndent(empty, "", "\t")
	renderer.Println(string(buf) + "\n")
}

func structMetaToArray(meta *toolbox.StructMeta) ([]string, [][]string) {
	var header = []string{"Name", "Type", "Required", "Description"}
	var data = make([][]string, 0)
	for _, field := range meta.Fields {

		data = append(data, []string{field.Name, field.Type, toolbox.AsString(field.Required), field.Description})
	}
	return header, data

}

func printStructMeta(renderer *endly.Renderer, color string, meta *toolbox.StructMeta) {
	header, data := structMetaToArray(meta)
	renderer.PrintTable(renderer.ColorText(meta.Type, color), header, data, 110)
	if len(meta.Dependencies) == 0 {
		return
	}
	for _, dep := range meta.Dependencies {
		printStructMeta(renderer, color, dep)
	}
}

func printServiceActionRequest() {

	service := endly.NewMetaService()

	var serviceID = flag.Lookup("s").Value.String()
	var action = flag.Lookup("a").Value.String()

	meta, err := service.Lookup(serviceID, action)
	if err != nil {
		log.Fatal(err)
	}
	var renderer = endly.NewRenderer(os.Stderr, 120)
	renderer.Println(renderer.ColorText("Request: ", "blue", "bold") + fmt.Sprintf("%T", meta.Request))
	printServiceActionInfo(renderer, meta.RequestInfo, "blue", "request", meta.Request)
	printStructMeta(renderer, "blue", meta.RequestMeta)
	renderer.Println(renderer.ColorText("Response: ", "green", "bold") + fmt.Sprintf("%T", meta.Response))
	printServiceActionInfo(renderer, meta.ResponseInfo, "green", "response", meta.Response)
	printStructMeta(renderer, "green", meta.ResponseMeta)
}

func printServiceActions() {
	manager := endly.NewManager()
	context := manager.NewContext(toolbox.NewContext())

	var serviceID = flag.Lookup("s").Value.String()

	if serviceID == "*" {
		fmt.Printf("endly services:\n")
		for k, v := range endly.Services(manager) {
			fmt.Printf("%v %T\n", k, v)
		}
		return
	}

	service, err := context.Service(serviceID)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("'%v' service actions: \n", serviceID)
	for _, action := range service.Actions() {
		route, _ := service.ServiceActionRoute(action)
		fmt.Printf("\t%v - %v\n", action, route.RequestInfo.Description)
	}
}

func getWorkflow(URL string) (*endly.Workflow, error) {
	dao := endly.NewWorkflowDao()
	manager := endly.NewManager()
	context := manager.NewContext(toolbox.NewContext())
	return dao.Load(context, url.NewResource(URL))
}

func printWorkflow(URL string) {
	workflow, err := getWorkflow(URL)
	if err != nil {
		log.Fatal(err)
	}
	printInFormat(workflow, "failed to print workflow: "+URL+", %v")

}

func printInFormat(source interface{}, errorTemplate string) {
	format := flag.Lookup("f").Value.String()
	var buf []byte
	var err error
	switch format {
	case "yaml":
		buf, err = yaml.Marshal(source)
	default:
		buf, err = json.MarshalIndent(source, "", "\t")
	}
	if err != nil {
		log.Fatalf(errorTemplate, err)
	}
	fmt.Printf("%s\n", buf)
}

func printHelp() {
	_, name := path.Split(os.Args[0])
	fmt.Fprintf(os.Stderr, "Usage of %s:\n", name)
	fmt.Fprintf(os.Stderr, "endly [options] [params...]\n")
	fmt.Fprintf(os.Stderr, "\tparams should be key value pair to be supplied as actual workflow parameters\n")
	fmt.Fprintf(os.Stderr, "\tif -r options is used, original request params may be overridden \n\n")

	fmt.Fprintf(os.Stderr, "where options include:\n")
	flag.PrintDefaults()
}

func printVersion() {
	fmt.Fprintf(os.Stdout, "%v %v\n", endly.AppName, endly.GetVersion())
}

func getRunRequestURL(candidate string) (*url.Resource, error) {
	if path.Ext(candidate) == "" {
		candidate = candidate + ".json"
	}
	resource := url.NewResource(candidate)
	if _, err := resource.Download(); err != nil {
		resource = url.NewResource(fmt.Sprintf("mem://%v/req/%v", endly.EndlyNamespace, candidate))
		if _, memError := resource.Download(); memError != nil {
			return nil, err
		}
	}
	return resource, nil

}

func getRunRequestWithOptions(flagset map[string]string) (*endly.WorkflowRunRequest, *endly.RunnerReportingOptions, error) {
	var request *endly.WorkflowRunRequest
	var options = &endly.RunnerReportingOptions{}
	if value, ok := flagset["w"]; ok {
		request = &endly.WorkflowRunRequest{
			WorkflowURL: value,
		}
		options = endly.DefaultRunnerReportingOption()
	}
	if value, ok := flagset["r"]; ok {
		resource, err := getRunRequestURL(value)
		if err == nil {
			request = &endly.WorkflowRunRequest{}
			err = resource.JSONDecode(request)
		}
		if request.WorkflowURL == "" {
			parent, _ := toolbox.URLSplit(resource.URL)
			parent = strings.Replace(parent, "req", "workflow", 1)
			request.WorkflowURL = toolbox.URLPathJoin(parent, request.Name+".csv")
		}
		if err != nil {
			return nil, nil, fmt.Errorf("failed to locate workflow run request: %v %v", value, err)
		}
		resource.JSONDecode(options)
		if options.Filter == nil {
			options.Filter = endly.DefaultRunnerReportingOption().Filter
		}
	}

	var params = endly.Pairs(getArguments()...)

	if request != nil {
		if len(request.Params) == 0 {
			request.Params = params
		}
		for k, v := range params {
			request.Params[k] = v
		}
		if value, ok := flagset["d"]; ok {
			request.EnableLogging = toolbox.AsBoolean(value)
			request.LoggingDirectory = flag.Lookup("l").Value.String()
		}
		if value, ok := flagset["t"]; ok {
			request.Tasks = value
		}
		if value, ok := flagset["i"]; ok {
			request.TagIDs = value
		}
	}
	return request, options, nil
}

func normalizeArgument(value string) interface{} {
	value = strings.Trim(value, " \"'")
	if strings.HasPrefix(value, "#") || strings.HasPrefix(value, "@") {
		resource := url.NewResource(string(value[1:]))
		text, err := resource.DownloadText()
		if err == nil {
			value = text
		}
	}
	if structure, err := toolbox.JSONToInterface(value); err == nil {
		return structure
	}
	return value
}

func getArguments() []interface{} {
	var arguments = make([]interface{}, 0)
	if len(os.Args) > 1 {
		for i := 1; i < len(os.Args); i++ {
			if strings.HasPrefix(os.Args[i], "-") {
				if !strings.Contains(os.Args[i], "=") {
					i++
				}
				continue
			}
			arguments = append(arguments, normalizeArgument(os.Args[i]))
		}
	}
	return arguments
}
