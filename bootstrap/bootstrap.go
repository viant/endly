package bootstrap


import (
	"encoding/json"
	"flag"
	"fmt"

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

	_ "github.com/alexbrainman/odbc"

	_ "github.com/viant/endly/gen/static"
	_ "github.com/viant/endly/shared/static" //load external resource like .csv .json files to mem storage

	_ "github.com/viant/endly/workflow"
	_ "github.com/viant/toolbox/storage/gs"
	_ "github.com/viant/toolbox/storage/s3"
	_ "github.com/viant/toolbox/storage/scp"

	_ "github.com/viant/endly/testing/dsunit"
	_ "github.com/viant/endly/testing/log"
	_ "github.com/viant/endly/testing/validator"

	_ "github.com/viant/endly/testing/endpoint/http"
	_ "github.com/viant/endly/testing/endpoint/smtp"
	_ "github.com/viant/endly/testing/msg"
	_ "github.com/viant/endly/testing/runner/http"
	_ "github.com/viant/endly/testing/runner/rest"
	_ "github.com/viant/endly/testing/runner/selenium"

	_ "github.com/viant/endly/deployment/build"
	_ "github.com/viant/endly/deployment/deploy"
	_ "github.com/viant/endly/deployment/sdk"
	_ "github.com/viant/endly/deployment/vc"

	_ "github.com/viant/endly/notify/smtp"

	_ "github.com/viant/endly/system/cloud/aws/apigateway"
	_ "github.com/viant/endly/system/cloud/aws/cloudwatch"
	_ "github.com/viant/endly/system/cloud/aws/dynamodb"
	_ "github.com/viant/endly/system/cloud/aws/ec2"
	_ "github.com/viant/endly/system/cloud/aws/iam"
	_ "github.com/viant/endly/system/cloud/aws/kinesis"
	_ "github.com/viant/endly/system/cloud/aws/lambda"
	_ "github.com/viant/endly/system/cloud/aws/logs"
	_ "github.com/viant/endly/system/cloud/aws/s3"
	_ "github.com/viant/endly/system/cloud/aws/ses"
	_ "github.com/viant/endly/system/cloud/aws/sns"
	_ "github.com/viant/endly/system/cloud/aws/sqs"

	_ "github.com/viant/endly/system/cloud/gc/bigquery"
	_ "github.com/viant/endly/system/cloud/gc/cloudfunctions"
	_ "github.com/viant/endly/system/cloud/gc/compute"
	_ "github.com/viant/endly/system/cloud/gc/pubsub"

	_ "github.com/viant/endly/system/daemon"
	_ "github.com/viant/endly/system/docker/ssh"
	_ "github.com/viant/endly/system/exec"
	_ "github.com/viant/endly/system/network"
	_ "github.com/viant/endly/system/process"
	_ "github.com/viant/endly/system/storage"

	"bufio"
	"errors"
	"github.com/viant/endly"
	"github.com/viant/endly/cli"
	"github.com/viant/endly/gen/web"
	"github.com/viant/endly/meta"
	"github.com/viant/endly/model"
	"github.com/viant/endly/workflow"
	"github.com/viant/toolbox"
	"github.com/viant/toolbox/cred"
	"github.com/viant/toolbox/data"
	"github.com/viant/toolbox/url"
	"golang.org/x/crypto/ssh/terminal"

	"github.com/google/gops/agent"
	rec "github.com/viant/endly/testing/endpoint/http"
	"gopkg.in/yaml.v2"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path"
	"strings"
	"syscall"
	"time"
)

func init() {

	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)

	flag.String("r", "run", "<path/url to workflow run request in YAML or JSON format>")
	flag.String("w", "manager", "<workflow name>  if both -r or -p and -w are specified, -w is ignored")
	flag.String("i", "", "<coma separated tagID list> to filter")

	flag.String("t", "*", "<task/s to run>, t='?' to list all tasks for selected workflow")

	flag.String("l", "logs", "<log directory>")
	flag.Bool("d", false, "enable logging")

	flag.Bool("p", false, "print workflow  as JSON or YAML")
	flag.String("f", "json", "<workflow or request format>, json or yaml")

	flag.Bool("h", false, "print help")
	flag.Bool("v", false, "print version")

	flag.String("s", "", "<serviceID> print service details, -s='*' prints all service IDs")
	flag.String("a", "", "<action> prints service action request/response detail")

	flag.String("c", "", "<credentials>, generate secret credentials file: ~/.secret/<credentials>.json")
	flag.String("k", "", "<private key path>,  works only with -c options, i.e -k="+path.Join(os.Getenv("HOME"), ".secret/id_rsa.pub"))

	flag.String("x", "", "xunit summary report format: xml|yaml|json")
	flag.Bool("g", false, "open test project generator")

	flag.String("u", "", "start HTTP recorder for the supplied URLs (testing/endpoint/http)")

	flag.Bool("m", false, "interactive mode (does not terminates process after workflow completes)")

	flag.Int("e", 5, "max number of failures CLI reported per validation, 0 - all failures reported")

	_ = mysql.SetLogger(&emptyLogger{})

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

	if toolbox.AsBoolean(flagset["g"]) {
		openTestGenerator()
		return
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
			delete(flagset, "r")
			flagset["w"] = flag.Lookup("w").Value.String()
			request, err = getRunRequestWithOptions(flagset)
			if err != nil && !strings.Contains(err.Error(), "no such file or directory") {
				log.Fatal(err)
			}
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

	runner := cli.New()
	err = runner.Run(request)
	if err != nil {
		log.Fatal(err)
	}

	interactive, ok := flagset["m"]
	if ok && toolbox.AsBoolean(interactive) {
		log.Printf("terminate by ctr-c\n")
		makeInteractive()
	}

	time.Sleep(time.Second)
}

func openbrowser(url string) {
	log.Printf("opening http://127.0.0.1:8071/ ...")
	exec.Command("open", url).Start()
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

func openTestGenerator() {

	baseURL := fmt.Sprintf("mem://%v", endly.Namespace)
	service := web.NewService(
		toolbox.URLPathJoin(baseURL, "template"),
		toolbox.URLPathJoin(baseURL, "asset"),
	)
	web.NewRouter(service, func(request *http.Request) {})
	go http.ListenAndServe(":8071", nil)
	time.Sleep(time.Second)
	openbrowser("http://127.0.0.1:8071/")
	makeInteractive()

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

func enableDiagnostics() {
	if err := agent.Listen(agent.Options{}); err != nil {
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
		log.Fatalf("failed to read password %v", err)
	}
	fmt.Print("\nRetype Password: ")
	bytePassword2, err := terminal.ReadPassword(int(syscall.Stdin))
	if err != nil {
		log.Fatalf("failed to read password %v", err)
	}

	password := string(bytePassword)
	if string(bytePassword2) != password {
		return "", "", errors.New("Password did not match")
	}
	return strings.TrimSpace(username), strings.TrimSpace(password), nil
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
	buf, _ = yaml.Marshal(req)
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
		route, _ := service.Route(action)
		fmt.Printf("\t%v - %v\n", action, route.RequestInfo.Description)
	}
}

func getWorkflow(request *workflow.RunRequest) (*model.Workflow, error) {
	if request.InlineWorkflow != nil && len(request.Pipeline) > 0 {
		baseURL, name := toolbox.URLSplit(request.AssetURL)
		name = strings.Replace(name, path.Ext(name), "", 1)
		return request.AsWorkflow(name, baseURL)
	}
	manager := endly.New()
	context := manager.NewContext(nil)
	var response = &workflow.LoadResponse{}
	var source = workflow.GetResource(workflow.NewDao(), context.State(), request.URL)
	if err := endly.Run(context, &workflow.LoadRequest{Source: source}, response); err != nil {
		return nil, err
	}
	return response.Workflow, nil
}

func printWorkflow(request *workflow.RunRequest) {
	workFlow, err := getWorkflow(request)
	if err != nil {
		log.Fatal(err)
	}
	printInFormat(workFlow, "failed to print workFlow: "+request.URL+", %v", true)

}

func printInFormat(source interface{}, errorTemplate string, hideEmpty bool) {
	if hideEmpty {
		var aMap = map[string]interface{}{}
		if err := toolbox.DefaultConverter.AssignConverted(&aMap, source); err == nil {
			mapSource := data.Map(toolbox.DeleteEmptyKeys(aMap))
			source = mapSource.AsEncodableMap()
		}
	}

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

func getRunRequestURL(URL string) (*url.Resource, error) {
	resource := url.NewResource(URL)
	var candidates = make([]string, 0)
	if path.Ext(resource.ParsedURL.Path) == "" {
		candidates = append(candidates, URL+".json", URL+".yaml")
	} else {
		candidates = append(candidates, URL)
	}

	var err error
	for _, candidate := range candidates {
		resource = url.NewResource(candidate)
		_, err = resource.Download()
		if err == nil {
			break
		}
		resource = url.NewResource(fmt.Sprintf("mem://%v/req/%v", endly.Namespace, candidate))
		if _, memError := resource.Download(); memError != nil {
			continue
		}
	}
	return resource, err
}

func getRunRequestWithOptions(flagset map[string]string) (*workflow.RunRequest, error) {
	var request *workflow.RunRequest
	var URL string
	if value, ok := flagset["w"]; ok {
		URL = value
		request = &workflow.RunRequest{
			URL: value,
		}
	}
	assetURL := ""
	if value, ok := flagset["r"]; ok {
		URL = value
		resource, err := getRunRequestURL(value)
		if err != nil {
			return nil, err
		}
		request = &workflow.RunRequest{}
		err = resource.Decode(request)
		if err != nil {
			return nil, fmt.Errorf("failed to locate workflow run request: %v %v", value, err)
		}
		assetURL = resource.URL
		request.Source = resource
		if request.Name == "" {
			request.Name = model.WorkflowSelector(URL).Name()
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
	request.AssetURL = assetURL
	err := request.Init()
	if value, ok := flagset["i"]; ok {
		request.TagIDs = value
	}
	if value, ok := flagset["e"]; ok {
		request.FailureCount = toolbox.AsInt(value)
	}
	updateBaseRunWithOptions(request, flagset)
	return request, err
}

func updateBaseRunWithOptions(request *workflow.RunRequest, flagset map[string]string) {
	var params = toolbox.Pairs(getArguments()...)
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
}

func normalizeArgument(value string) interface{} {
	value = strings.Trim(value, " \"'")
	if strings.HasPrefix(value, "#") || strings.HasPrefix(value, "@") {
		resource := url.NewResource(string(value[1:]))
		var dataStructure = map[string]interface{}{}
		if err := resource.Decode(&dataStructure); err == nil {
			return dataStructure
		}
		if text, err := resource.DownloadText(); err == nil {
			return text
		}
	}
	return value
}

func getArguments() []interface{} {
	var arguments = make([]interface{}, 0)
	if len(os.Args) > 1 {
		for i := 1; i < len(os.Args); i++ {
			var candidate = os.Args[i]
			if strings.HasPrefix(candidate, "-") {
				if !strings.Contains(candidate, "=") {
					i++
				}
				continue
			}
			keyValuePair := strings.SplitN(candidate, "=", 2)
			if len(keyValuePair) == 2 {
				arguments = append(arguments, keyValuePair[0], normalizeArgument(keyValuePair[1]))
			} else {
				arguments = append(arguments, normalizeArgument(candidate))
			}
			arguments = append(arguments)
		}
	}
	return arguments
}

func startRecorder(URLs []string) {
	rec.StartRecorder(URLs...)
}

type emptyLogger struct{}

func (l *emptyLogger) Print(v ...interface{}) {

}
