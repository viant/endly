package bootstrap

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/viant/afs"
	"github.com/viant/endly/model/location"
	"github.com/viant/endly/service/meta"
	"github.com/viant/scy"
	"sort"

	"github.com/viant/endly/internal/util"

	//Database/datastore dependencies

	_ "github.com/MichaelS11/go-cql-driver"
	"github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	_ "github.com/viant/asc"
	_ "github.com/viant/bgc"

	_ "github.com/adrianwit/dyndb"
	_ "github.com/adrianwit/fbc"
	_ "github.com/adrianwit/fsc"
	_ "github.com/adrianwit/mgc"

	_ "github.com/viant/afsc/aws"
	_ "github.com/viant/afsc/gcp"
	_ "github.com/viant/endly/service/system/secret"
	"github.com/viant/scy/cred/secret/term"
	_ "github.com/viant/scy/kms/blowfish"
	_ "github.com/viant/scy/kms/gcp"

	//cgo _ "github.com/alexbrainman/odbc"
	//cgo _"github.com/mattn/go-oci8"

	_ "github.com/viant/endly/service/shared" //load external resource like .csv .json files to mem storage

	_ "github.com/viant/endly/service/migrator"
	_ "github.com/viant/endly/service/workflow"
	_ "github.com/viant/toolbox/storage/gs"
	_ "github.com/viant/toolbox/storage/s3"
	_ "github.com/viant/toolbox/storage/scp"

	_ "github.com/viant/endly/service/testing/dsunit"
	_ "github.com/viant/endly/service/testing/log"
	_ "github.com/viant/endly/service/testing/validator"

	_ "github.com/viant/endly/service/testing/endpoint/http"
	_ "github.com/viant/endly/service/testing/endpoint/smtp"
	_ "github.com/viant/endly/service/testing/msg"
	_ "github.com/viant/endly/service/testing/runner/http"
	_ "github.com/viant/endly/service/testing/runner/rest"
	_ "github.com/viant/endly/service/testing/runner/webdriver"

	_ "github.com/viant/endly/service/deployment/build"
	_ "github.com/viant/endly/service/deployment/deploy"
	_ "github.com/viant/endly/service/deployment/sdk"
	_ "github.com/viant/endly/service/deployment/vc"
	_ "github.com/viant/endly/service/deployment/vc/git"

	_ "github.com/viant/endly/service/notify/slack"
	_ "github.com/viant/endly/service/notify/smtp"

	_ "github.com/viant/endly/service/system/cloud/aws/apigateway"
	_ "github.com/viant/endly/service/system/cloud/aws/cloudwatch"
	_ "github.com/viant/endly/service/system/cloud/aws/cloudwatchevents"

	_ "github.com/viant/endly/service/system/cloud/aws/dynamodb"
	_ "github.com/viant/endly/service/system/cloud/aws/ec2"
	_ "github.com/viant/endly/service/system/cloud/aws/iam"
	_ "github.com/viant/endly/service/system/cloud/aws/kinesis"
	_ "github.com/viant/endly/service/system/cloud/aws/kms"
	_ "github.com/viant/endly/service/system/cloud/aws/lambda"
	_ "github.com/viant/endly/service/system/cloud/aws/logs"
	_ "github.com/viant/endly/service/system/cloud/aws/rds"
	_ "github.com/viant/endly/service/system/cloud/aws/s3"
	_ "github.com/viant/endly/service/system/cloud/aws/ses"
	_ "github.com/viant/endly/service/system/cloud/aws/sns"
	_ "github.com/viant/endly/service/system/cloud/aws/sqs"
	_ "github.com/viant/endly/service/system/cloud/aws/ssm"

	_ "github.com/viant/endly/service/system/cloud/gcp/bigquery"
	_ "github.com/viant/endly/service/system/cloud/gcp/cloudfunctions"
	_ "github.com/viant/endly/service/system/cloud/gcp/cloudscheduler"
	_ "github.com/viant/endly/service/system/cloud/gcp/compute"
	_ "github.com/viant/endly/service/system/cloud/gcp/container"
	_ "github.com/viant/endly/service/system/cloud/gcp/kms"
	_ "github.com/viant/endly/service/system/cloud/gcp/pubsub"
	_ "github.com/viant/endly/service/system/cloud/gcp/run"
	_ "github.com/viant/endly/service/system/cloud/gcp/storage"

	_ "github.com/viant/endly/service/system/daemon"
	_ "github.com/viant/endly/service/system/docker"
	_ "github.com/viant/endly/service/system/exec"
	_ "github.com/viant/endly/service/system/process"
	_ "github.com/viant/endly/service/system/storage"

	"github.com/viant/endly"
	"github.com/viant/endly/cli"
	"github.com/viant/endly/model"
	"github.com/viant/endly/service/workflow"
	"github.com/viant/scy/cred"
	"github.com/viant/toolbox"
	"github.com/viant/toolbox/data"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"path"
	"strings"
	"syscall"
	"time"

	"github.com/google/gops/agent"
	rec "github.com/viant/endly/service/testing/endpoint/http"
	"gopkg.in/yaml.v3"
)

func init() {

	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)

	flag.String("r", "run", "<path/url to workflow run request in YAML or JSON format>")
	flag.String("i", "", "<coma separated tagID list> to filter")

	flag.String("t", "*", "<task/s to run>, t='?' to list all tasks for selected workflow")

	flag.String("l", "logs", "<log directory>")
	flag.Bool("d", false, "enable logging")

	flag.Bool("p", false, "print workflow  as JSON or YAML")
	flag.String("f", "json", "<workflow or request format>, json or yaml")

	flag.Bool("h", false, "print help")
	flag.Bool("v", false, "print version")

	flag.Bool("j", false, "list user defined function (UDF)")
	flag.String("s", "", "<serviceID> print service details, -s='*' prints all service IDs")
	flag.String("a", "", "<action> prints service action request/response detail")

	flag.String("c", "", "<credentials>, generate secret credentials file: ~/.secret/<credentials>.json")
	flag.String("k", "", "<private key path>,  works only with -c options, i.e -k="+path.Join(os.Getenv("HOME"), ".secret/id_rsa"))
	flag.String("endpoint", "", "<endpoint for generated secrets credentials>,  works only with -c options, i.e -endpoint=127.0.0.1")

	flag.String("x", "", "xunit summary report format: xml|yaml|json")
	flag.Bool("g", false, "open test project generator")

	flag.String("u", "", "start HTTP recorder for the supplied URLs (testing/endpoint/http)")
	flag.Bool("m", false, "interactive mode (does not terminates process after workflow completes)")
	flag.Int("e", 5, "max number of failures CLI reported per validation, 0 - all failures reported")
	flag.String("run", "", "run specified service action it expect valid service:action to run")
	flag.String("req", "", "optional request URL when run option is specified")
	_ = mysql.SetLogger(&emptyLogger{})

}

func detectFirstArguments(flagset map[string]string) {
	if len(os.Args) == 1 {
		return
	}
	candidate := os.Args[1]
	if strings.Contains(candidate, "=") {
		return
	}
	if strings.Contains(candidate, ":") {
		flagset["run"] = os.Args[1]
	} else {
		if strings.Contains(candidate, ".") {
			flagset["r"] = os.Args[1]
		} else if toolbox.FileExists(fmt.Sprintf("%v.yaml", candidate)) {
			flagset["r"] = fmt.Sprintf("%v.yaml", candidate)
		} else {
			return
		}
	}
	if len(os.Args) > 2 {
		os.Args = append([]string{os.Args[0]}, os.Args[2:]...)
	}
}

func Bootstrap() {

	flagset := make(map[string]string)
	flag.Usage = printHelp

	detectFirstArguments(flagset)
	flag.Parse()

	flag.Visit(func(f *flag.Flag) {
		if f.Value.String() != "" {
			flagset[f.Name] = f.Value.String()
		}
	})
	_, shouldQuit := flagset["v"]
	flagset["v"] = flag.Lookup("v").Value.String()

	if URLs, ok := flagset["u"]; ok {
		startRecorder(strings.Split(URLs, " "))
		return
	}

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

	if _, ok := flagset["j"]; ok {
		printUDFs()
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

	if run, ok := flagset["run"]; ok {
		err := runAction(context.Background(), run, flagset)
		if err != nil {
			log.Fatal(err)
		}
		return
	}

	request, err := getRunRequestWithOptions(flagset)
	if err != nil {
		log.Fatal(err)
	}
	if request == nil {
		flagset["r"] = flag.Lookup("r").Value.String()
		request, err = getRunRequestWithOptions(flagset)
		if err != nil && !strings.Contains(err.Error(), "no such file or directory") {
			log.Fatal(err)
		}

		if request == nil {
			request, err = getRunRequestWithOptions(flagset)
		}
	}
	if request == nil {
		printHelp()
		return
	}
	if value, ok := flagset["p"]; ok && toolbox.AsBoolean(value) {
		printWorkflow(request)
		return
	}
	if flagset["t"] == "?" {
		printWorkflowTasks(request)
		return
	}
	interactive, ok := flagset["m"]
	runWorkflow(request, ok && toolbox.AsBoolean(interactive))
}

func runAction(ctx context.Context, run string, flagset map[string]string) error {
	request, err := loadInlineWorkflow(ctx, "mem://github.com/viant/endly/service/workflow/adhoc.yaml")
	if err != nil {
		return err
	}
	baseURL, _ := toolbox.URLSplit(request.AssetURL)
	currentURL := location.NewResource("").URL
	argsMap, err := util.GetArguments(currentURL, baseURL)
	if err != nil {
		return err
	}
	if req, ok := flagset["req"]; ok {
		requestData, err := util.LoadData([]string{baseURL}, req)
		if err != nil {
			return err
		}
		reqMap := data.Map(argsMap)
		argsMap = toolbox.AsMap(reqMap.Expand(requestData))
	}

	request.Inlined.State = data.NewMap()
	request.Inlined.State.Put("run", run)
	request.Inlined.State.Put("request", argsMap)
	err = updateBaseRunWithOptions(request, flagset)
	if err != nil {
		return err
	}
	if value, ok := flagset["p"]; ok && toolbox.AsBoolean(value) {
		printWorkflow(request)
		return nil
	}
	interactive, ok := flagset["m"]
	runWorkflow(request, ok && toolbox.AsBoolean(interactive))
	return nil
}

func runWorkflow(request *workflow.RunRequest, interactive bool) {
	runner := cli.New()
	request.Interactive = interactive
	err := runner.Run(request)
	if err != nil {
		log.Fatal(err)
	}
	if interactive {
		log.Printf("terminate by ctr-c\n")
		makeInteractive()
	}
	time.Sleep(time.Second)
}

func printUDFs() {
	manager := endly.New()
	context := manager.NewContext(nil)
	state := context.State()
	var udfs = make([]string, 0)
	for k, v := range state {
		if toolbox.IsFunc(v) {
			udfs = append(udfs, k)
		}
	}
	sort.Strings(udfs)
	fmt.Printf("User defined functions:\n")
	for _, name := range udfs {
		fmt.Printf("\t$%v()\n", name)
	}

}

func openbrowser(url string) {
	log.Printf("opening http://127.0.0.1:8071/ ...")
	_ = exec.Command("open", url).Start()
}

func makeInteractive() {
	signal_chan := make(chan os.Signal, 1)
	signal.Notify(signal_chan,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT)
	exit_chan := make(chan int)
	go func() {
		for {
			s := <-signal_chan
			switch s {
			// kill -SIGHUP XXXX
			case syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT:
				exit_chan <- 0
			}
		}
	}()
	code := <-exit_chan
	os.Exit(code)
}

func generateSecret(credentialsFile string) {
	secretPath := path.Join(os.Getenv("HOME"), ".secret")
	if !toolbox.FileExists(secretPath) {
		os.Mkdir(secretPath, 0744)
	}
	username, password, err := term.ReadUserAndPassword(term.ReadingCredentialTimeout)
	if err != nil {
		fmt.Printf("\n")
		log.Fatal(err)
	}
	if password == "" {
		fmt.Printf("password was empty")
		return
	}
	fmt.Println("")
	genericCred := &cred.Generic{
		SSH: cred.SSH{
			Basic: cred.Basic{
				Username: username,
				Password: password,
			},
		},
	}

	var privateKeyPath = flag.Lookup("k").Value.String()
	privateKeyPath = strings.Replace(privateKeyPath, "~", os.Getenv("HOME"), 1)
	if endpoint := flag.Lookup("endpoint"); endpoint != nil {
		genericCred.Endpoint = endpoint.Value.String()
	}
	if privateKeyPath != "" && !toolbox.FileExists(privateKeyPath) {
		log.Fatalf("unable to locate private key: %v \n", privateKeyPath)
	}
	genericCred.PrivateKeyPath = privateKeyPath
	var secretFile = path.Join(secretPath, fmt.Sprintf("%v.json", credentialsFile))
	scyService := scy.New()
	resource := scy.NewResource(genericCred, secretFile, "blowfish://default")
	secret := scy.NewSecret(genericCred, resource)
	if err = scyService.Store(context.Background(), secret); err != nil {
		fmt.Printf("\n")
		log.Fatal(err)
	}
}

func enableDiagnostics() {
	if err := agent.Listen(agent.Options{}); err != nil {
		log.Fatal(err)
	}
}

func printWorkflowTasks(request *workflow.RunRequest) {
	workFlow, err := getWorkflow(request)
	if err != nil {
		log.Fatal(err)
	}
	_, _ = fmt.Fprintf(os.Stderr, "Workflow '%v' (%v) tasks:\n", workFlow.Name, workFlow.Source.URL)
	for _, task := range workFlow.Tasks {
		_, _ = fmt.Fprintf(os.Stderr, "\t%v: %v\n", task.Name, task.Description)
	}
}

func requestName(name string, ext string) string {
	name = path.Ext(name)
	name = strings.ToLower(string(name[1:]))
	name = strings.Replace(name, "request", "", 1)
	return fmt.Sprintf("@%v.%v\n", name, ext)
}

func printServiceActionInfo(renderer *cli.Renderer, info *endly.ActionInfo, color, infoType string, req interface{}) {
	if info != nil {
		if info.Description != "" {
			renderer.Printf(renderer.ColorText("Description: ", color, "bold")+" %v\n", info.Description)
		}
		if len(info.Examples) > 0 {
			for i, example := range info.Examples {
				renderer.Printf(renderer.ColorText(fmt.Sprintf("Example %v: ", i+1), color, "bold")+" %v %v\n", example.Description, infoType)
				aMap, err := toolbox.JSONToMap(example.Data)

				if err == nil {
					buf, _ := json.MarshalIndent(aMap, "", "\t")

					renderer.Printf(requestName(fmt.Sprintf("%T", req), "json"))
					renderer.Println(string(buf))
					text, err := toolbox.AsYamlText(aMap)
					if err == nil {
						renderer.Printf(requestName(fmt.Sprintf("%T", req), "yaml"))
						renderer.Println(text)
					}

				} else {
					renderer.Printf("%v\n", example.Data)
				}

			}
		}
	}
	renderer.Printf(renderer.ColorText(fmt.Sprintf("JSON %v: \n", infoType), color, "bold"))
	buf, _ := json.MarshalIndent(req, "", "\t")
	renderer.Println(string(buf) + "\n")
	renderer.Printf(renderer.ColorText(fmt.Sprintf("YAML %v: \n", infoType), color, "bold"))
	reqMap := map[string]interface{}{}
	_ = toolbox.DefaultConverter.AssignConverted(&reqMap, req)
	if reqMap != nil {
		reqMap = toLowerCaseCamel(reqMap)
	}
	buf, _ = yaml.Marshal(reqMap)
	renderer.Println(string(buf) + "\n")
}

func toLowerCaseCamel(req map[string]interface{}) map[string]interface{} {
	var result = make(map[string]interface{})
	for k, v := range req {
		k = toolbox.ToCaseFormat(k, toolbox.CaseUpperCamel, toolbox.CaseLowerCamel)
		if v == nil {
			result[k] = v
			continue
		}
		if toolbox.IsMap(v) {
			v = toLowerCaseCamel(toolbox.AsMap(v))
		} else if toolbox.IsSlice(v) {
			aSlice := toolbox.AsSlice(v)
			for i := range aSlice {
				if aSlice[i] != nil && toolbox.IsMap(aSlice[i]) {
					aSlice[i] = toLowerCaseCamel(toolbox.AsMap(aSlice[i]))
				}
			}
			v = aSlice
		}
		result[k] = v
	}
	return result
}

func structMetaToArray(meta *toolbox.StructMeta) ([]string, [][]string) {
	var header = []string{"Name", "Type", "Required", "Description"}
	var data = make([][]string, 0)
	for _, field := range meta.Fields {

		data = append(data, []string{field.Name, field.Type, toolbox.AsString(field.Required), field.Description})
	}
	return header, data

}

func printStructMeta(renderer *cli.Renderer, color string, meta *toolbox.StructMeta) {
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

	service := meta.New()

	var serviceID = flag.Lookup("s").Value.String()
	var action = flag.Lookup("a").Value.String()

	meta, err := service.Lookup(serviceID, action)
	if err != nil {
		log.Fatal(err)
	}
	var renderer = cli.NewRenderer(os.Stderr, 120)
	renderer.Println(renderer.ColorText("ServiceRequest: ", "blue", "bold") + fmt.Sprintf("%T", meta.Request))
	printServiceActionInfo(renderer, meta.RequestInfo, "blue", "request", meta.Request)
	printStructMeta(renderer, "blue", meta.RequestMeta)
	renderer.Println(renderer.ColorText("Response: ", "green", "bold") + fmt.Sprintf("%T", meta.Response))
	printServiceActionInfo(renderer, meta.ResponseInfo, "green", "response", meta.Response)
	printStructMeta(renderer, "green", meta.ResponseMeta)
}

func printServiceActions() {
	manager := endly.New()
	context := manager.NewContext(toolbox.NewContext())

	var serviceID = flag.Lookup("s").Value.String()

	if strings.Contains(serviceID, ":") {
		pair := strings.SplitN(serviceID, ":", 2)
		_ = flag.CommandLine.Set("s", pair[0])
		_ = flag.CommandLine.Set("a", pair[1])
		printServiceActionRequest()
		return
	}

	if serviceID == "*" {
		services := endly.Services(manager)
		fmt.Printf("endly services:\n")
		var ids = make([]string, 0)
		for k := range services {
			ids = append(ids, k)
		}
		sort.Strings(ids)
		for _, k := range ids {
			fmt.Printf("%v %T\n", k, services[k])
		}
		return
	}

	service, err := context.Service(serviceID)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("'%v' service actions: \n", serviceID)
	for _, action := range service.Actions() {
		route, _ := service.Route(action)
		fmt.Printf("\t%v - %v\n", action, route.RequestInfo.Description)
	}
}

func getWorkflow(request *workflow.RunRequest) (*model.Workflow, error) {
	if request.Inlined != nil && len(request.Pipeline) > 0 {
		baseURL, name := toolbox.URLSplit(request.AssetURL)
		name = strings.Replace(name, path.Ext(name), "", 1)
		return request.AsWorkflow(name, baseURL)
	}
	return nil, fmt.Errorf("only yaml workflow are supported")
}

func printWorkflow(request *workflow.RunRequest) {
	workFlow, err := getWorkflow(request)
	if err != nil {
		log.Fatal(err)
	}
	printInFormat(workFlow, "failed to print workFlow: "+request.URL+", %v", true)

}

func printInFormat(source interface{}, errorTemplate string, hideEmpty bool) {
	//if hideEmpty {
	//	var aMap = map[string]interface{}{}
	//	if err := toolbox.DefaultConverter.AssignConverted(&aMap, source); err == nil {
	//		mapSource := data.Map(toolbox.DeleteEmptyKeys(aMap))
	//		source = mapSource.AsEncodableMap()
	//	}
	//}

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

func getRunRequestURL(ctx context.Context, URL string) (*location.Resource, error) {
	resource := location.NewResource(URL)
	var candidates = make([]string, 0)
	if path.Ext(resource.Path()) == "" {
		candidates = append(candidates, URL+".json", URL+".yaml")
	} else {
		candidates = append(candidates, URL)
	}
	fs := afs.New()

	var err error
	for _, candidate := range candidates {
		resource = location.NewResource(candidate)
		if ok, _ := fs.Exists(ctx, resource.URL); ok {
			break
		}
	}
	return resource, err
}

func getRunRequestWithOptions(flagset map[string]string) (*workflow.RunRequest, error) {
	var request *workflow.RunRequest
	var err error
	if value, ok := flagset["r"]; ok {
		if path.Ext(value) == "" {
			value += ".yaml"
		}
		if request, err = loadInlineWorkflow(context.Background(), value); err != nil {
			return nil, err
		}
	}

	if request == nil {
		return nil, nil
	}
	if value, ok := flagset["t"]; ok {
		request.Tasks = value
	}
	if value, ok := flagset["x"]; ok {
		request.SummaryFormat = value
	}
	err = request.Init()
	if value, ok := flagset["i"]; ok {
		request.TagIDs = value
	}

	if err == nil {
		err = updateBaseRunWithOptions(request, flagset)
	}
	return request, err
}

func loadInlineWorkflow(ctx context.Context, URL string) (*workflow.RunRequest, error) {
	resource, err := getRunRequestURL(ctx, URL)
	if err != nil {
		return nil, err
	}

	request := &workflow.RunRequest{}
	err = resource.Decode(request)
	if err != nil {
		return nil, fmt.Errorf("failed to locate workflow run request: %v %v", URL, err)
	}

	request.Source = resource
	if request.Name == "" {
		request.Name = model.WorkflowSelector(URL).Name()
	}
	request.AssetURL = resource.URL
	return request, err
}

func updateBaseRunWithOptions(request *workflow.RunRequest, flagset map[string]string) error {
	currentPath := location.NewResource("")
	parentURL, _ := toolbox.URLSplit(currentPath.URL)
	if request.Source != nil {
		parentURL, _ = toolbox.URLSplit(request.Source.URL)
	}
	params, err := util.GetArguments(currentPath.URL, parentURL)
	if err != nil {
		return err
	}
	if request != nil {
		if len(request.Params) == 0 {
			request.Params = params
		}
		for k, v := range params {
			request.Params[k] = v
		}
		if value, ok := flagset["d"]; ok {
			go enableDiagnostics()
			request.EnableLogging = toolbox.AsBoolean(value)
			request.LogDirectory = flag.Lookup("l").Value.String()
		}
	}
	if value, ok := flagset["e"]; ok {
		request.FailureCount = toolbox.AsInt(value)
	}
	return nil
}

func startRecorder(URLs []string) {
	rec.StartRecorder(URLs...)
}

type emptyLogger struct{}

func (l *emptyLogger) Print(v ...interface{}) {

}
