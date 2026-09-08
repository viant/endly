package android

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/viant/assertly"
	"github.com/viant/endly"
	"github.com/viant/endly/service/testing/runner/internal/mobile"
)

func (s *service) test(ctx *endly.Context, request *TestRequest) (*TestResponse, error) {
	if err := s.validateLease(request.Lease); err != nil {
		return nil, err
	}
	adb, err := resolveAndroidTool("adb", request.Lease.AndroidSDKRoot)
	if err != nil {
		return nil, err
	}
	if request.AppAPKPath != "" {
		for _, APK := range []string{request.AppAPKPath, request.TestAPKPath} {
			args := []string{"-s", request.Lease.Serial, "install", "-r", "-t"}
			if request.GrantAll {
				args = append(args, "-g")
			}
			args = append(args, APK)
			if result, err := s.runner.Run(ctx.Background(), mobile.Command{Name: adb, Args: args}); err != nil {
				return nil, fmt.Errorf("install instrumentation APK %s: %s: %w", APK, strings.TrimSpace(result.Stderr), err)
			}
		}
	}
	args := []string{"-s", request.Lease.Serial, "shell", "am", "instrument", "-w", "-r"}
	if request.Class != "" {
		args = append(args, "-e", "class", request.Class)
	}
	keys := make([]string, 0, len(request.Arguments))
	for key := range request.Arguments {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		args = append(args, "-e", key, request.Arguments[key])
	}
	args = append(args, request.TestPackage+"/"+request.Runner)
	testCtx, cancel := context.WithTimeout(ctx.Background(), time.Duration(request.TimeoutMs)*time.Millisecond)
	defer cancel()
	started := time.Now()
	result, runErr := s.runner.Run(testCtx, mobile.Command{Name: adb, Args: args})
	response := parseInstrumentation(result.Stdout + result.Stderr)
	response.DurationMs = int(time.Since(started) / time.Millisecond)
	if runErr != nil {
		return response, fmt.Errorf("run Android instrumentation: %w", runErr)
	}
	return response, nil
}

func parseInstrumentation(output string) *TestResponse {
	response := &TestResponse{Cases: []InstrumentationCase{}, Output: output, Validations: []*assertly.Validation{}}
	status := map[string]string{}
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "INSTRUMENTATION_STATUS: ") {
			pair := strings.SplitN(strings.TrimPrefix(line, "INSTRUMENTATION_STATUS: "), "=", 2)
			if len(pair) == 2 {
				status[pair[0]] = pair[1]
			}
			continue
		}
		if !strings.HasPrefix(line, "INSTRUMENTATION_STATUS_CODE: ") {
			continue
		}
		code, _ := strconv.Atoi(strings.TrimSpace(strings.TrimPrefix(line, "INSTRUMENTATION_STATUS_CODE: ")))
		if code == 1 {
			continue
		}
		item := InstrumentationCase{Class: status["class"], Name: status["test"], Stack: status["stack"]}
		switch code {
		case 0:
			item.Status = "passed"
			response.Passed++
		case -3, -4:
			item.Status = "skipped"
			response.Skipped++
		default:
			item.Status = "failed"
			response.Failed++
		}
		response.Cases = append(response.Cases, item)
		status = map[string]string{}
	}
	validation := assertly.NewValidation()
	validation.Description = "Android instrumentation"
	validation.PassedCount = response.Passed
	for _, item := range response.Cases {
		if item.Status != "failed" {
			continue
		}
		failure := assertly.NewFailure("android:test", item.Class+"."+item.Name, assertly.EqualViolation, "passed", item.Status)
		if item.Stack != "" {
			failure.Message += ": " + item.Stack
		}
		validation.AddFailure(failure)
	}
	response.Validations = append(response.Validations, validation)
	return response
}
