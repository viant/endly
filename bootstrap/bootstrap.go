package bootstrap

import (
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	_ "github.com/viant/asc"
	_ "github.com/viant/bgc"
	_ "github.com/viant/endly/static" //load external resource like .csv .json files to mem storage
	_ "github.com/viant/toolbox/storage/aws"
	_ "github.com/viant/toolbox/storage/gs"

	"flag"
	"fmt"
	"github.com/viant/endly"
	"github.com/viant/toolbox"
	"github.com/viant/toolbox/url"
	"log"
	"os"
	"path"
	"strings"
	"time"
	"encoding/json"
)

func init() {

	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
	flag.String("r", "run.json", "<path/url to workflow run request in JSON format> ")
	flag.String("w", "manager", "<workflow name>  if both -r and -w valid options are specified, -w is ignored")
	flag.String("t", "*", "<task/s to run>")
	flag.String("l", "logs", "<log directory>")
	flag.Bool("d", false, "enable logging")
	flag.Bool("p", false, "print neatly workflow as JSON")
	flag.Bool("h", false, "print help")
	flag.Bool("v", true, "print version")

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
	request, option, err := getRunRequestWithOptons(flagset)
	if request == nil {
		flagset["r"] = flag.Lookup("r").Value.String()
		flagset["w"] = flag.Lookup("w").Value.String()
		request, option, err = getRunRequestWithOptons(flagset)
	}

	if err != nil {
		log.Fatal(err)
	}
	if value, ok := flagset["p"]; ok && toolbox.AsBoolean(value) {
		printWorkflow(request.WorkflowURL)
		return
	}

	runner := endly.NewCliRunner()
	err = runner.Run(request, option)
	if err != nil {
		log.Fatal(err)
	}
	time.Sleep(time.Second)
}

func printWorkflow(URL string) {
	dao := endly.NewWorkflowDao()
	manager := endly.NewManager()
	context := manager.NewContext(toolbox.NewContext())
	if workflow, _ := dao.Load(context, url.NewResource(URL)); workflow != nil {

		buf, err := json.MarshalIndent(workflow, "", "\t")
		if err != nil {
			log.Fatal("failed to load workflow: %v, %v", URL, err)
		}
		fmt.Printf("%s", buf)
	}
}

func printHelp() {
	_, name := path.Split(os.Args[0])
	fmt.Fprintf(os.Stderr, "Usage of %s:\n", name)
	fmt.Fprintf(os.Stderr, "endly [options] [params...]\n")
	fmt.Fprintf(os.Stderr, "\tparams should be key value pair to be supplied as actual workflow parameters\n")
	fmt.Fprintf(os.Stderr, "\tif -r options is used, original request params may be overriden\n\n")

	fmt.Fprintf(os.Stderr, "where options include:\n")
	flag.PrintDefaults()
}

func printVersion() {
	fmt.Fprintf(os.Stdout, "%v %v\n", endly.AppName, endly.GetVersion())
}

func getWorkflowURL(candidate string) (string, string, error) {
	var _,  name = path.Split(candidate)
	if path.Ext(candidate) == "" {
		candidate = candidate + ".csv"
	} else {
		name = string(name[:len(name)-4]) //remove extension
	}
	resource := url.NewResource(candidate)
	if _, err := resource.Download(); err != nil {
		resource = url.NewResource(fmt.Sprintf("mem://%v/workflow/%v", endly.EndlyNamespace, candidate))
		if _, memError := resource.Download(); memError != nil {
			return "", "", err
		}
	}
	return name, resource.URL, nil
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

func getRunRequestWithOptons(flagset map[string]string) (*endly.WorkflowRunRequest, *endly.RunnerReportingOptions, error) {
	var request *endly.WorkflowRunRequest
	var options = &endly.RunnerReportingOptions{}
	if value, ok := flagset["w"]; ok {
		name, URL, err := getWorkflowURL(value)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to locate workflow: %v %v", value, err)
		}
		request = &endly.WorkflowRunRequest{
			WorkflowURL: URL,
			Name:        name,
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
	}
	return request, options, nil
}

func normalizeArgument(value string) interface{} {
	value = strings.Trim(value, " \"'")
	if strings.HasPrefix(value, "#") {
		resource := url.NewResource(value)
		if text, err := resource.DownloadText(); err == nil {
			value = text
		}
	}
	_, structure := endly.AsExtractable(value)
	if len(structure) > 0 {
		return structure
	}
	return value
}

func getArguments() []interface{} {
	var arguments = make([]interface{}, 0)
	if len(os.Args) > 1 {
		for i := 1; i < len(os.Args); i++ {
			if strings.HasPrefix(os.Args[i], "-") {
				if ! strings.Contains(os.Args[i], "=") {
					i++
				}
				continue
			}
			arguments = append(arguments, normalizeArgument(os.Args[i]))
		}
	}
	return arguments
}
