package ios

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/viant/assertly"
	"github.com/viant/endly"
	"github.com/viant/endly/service/testing/runner/internal/mobile"
)

func (s *service) test(ctx *endly.Context, request *TestRequest) (*TestResponse, error) {
	if err := s.validateLease(request.Destination); err != nil {
		return nil, err
	}
	xcodebuild, err := mobile.ResolveExecutable("xcodebuild", "/usr/bin/xcodebuild")
	if err != nil {
		return nil, err
	}
	args := []string{}
	if request.Mode == "xctestrun" {
		args = append(args, "-xctestrun", request.XCTestRunPath)
	} else if request.WorkspacePath != "" {
		args = append(args, "-workspace", request.WorkspacePath)
	} else {
		args = append(args, "-project", request.ProjectPath)
	}
	if request.Mode != "xctestrun" {
		args = append(args, "-scheme", request.Scheme)
	}
	collectDiagnostics := request.CollectDiagnostics
	if collectDiagnostics == "" {
		collectDiagnostics = "never"
	}
	args = append(args,
		"-destination", "platform=iOS Simulator,id="+request.Destination.UDID,
		"-resultBundlePath", request.ResultBundlePath,
		"-collect-test-diagnostics", collectDiagnostics,
	)
	if request.DerivedDataPath != "" {
		args = append(args, "-derivedDataPath", request.DerivedDataPath)
	}
	for _, identifier := range request.OnlyTesting {
		args = append(args, "-only-testing:"+identifier)
	}
	for _, identifier := range request.SkipTesting {
		args = append(args, "-skip-testing:"+identifier)
	}
	if request.Mode == "scheme" {
		args = append(args, "test")
	} else {
		args = append(args, "test-without-building")
	}
	testCtx, cancel := context.WithTimeout(ctx.Background(), time.Duration(request.TimeoutMs)*time.Millisecond)
	defer cancel()
	started := time.Now()
	result, runErr := s.runner.Run(testCtx, mobile.Command{Name: xcodebuild, Args: args})
	response := &TestResponse{Output: result.Stdout + result.Stderr, DurationMs: int(time.Since(started) / time.Millisecond)}
	xcrun, resolveErr := mobile.ResolveExecutable("xcrun", "/usr/bin/xcrun")
	if resolveErr != nil {
		if runErr != nil {
			return response, fmt.Errorf("run XCTest: %w", runErr)
		}
		return response, resolveErr
	}
	summaryResult, summaryErr := s.runner.Run(testCtx, mobile.Command{Name: xcrun, Args: []string{"xcresulttool", "get", "test-results", "summary", "--path", request.ResultBundlePath, "--compact"}})
	if summaryErr != nil {
		if runErr != nil {
			return response, fmt.Errorf("run XCTest: %w; read xcresult: %v", runErr, summaryErr)
		}
		return response, fmt.Errorf("read xcresult summary: %w", summaryErr)
	}
	if err := json.Unmarshal([]byte(summaryResult.Stdout), &response.Summary); err != nil {
		return response, fmt.Errorf("decode xcresult summary: %w", err)
	}
	response.Validations = []*assertly.Validation{xcTestValidation(response.Summary)}
	if runErr != nil && response.Summary.FailedTests == 0 {
		return response, fmt.Errorf("run XCTest: %w", runErr)
	}
	return response, nil
}

func xcTestValidation(summary XCTestSummary) *assertly.Validation {
	validation := assertly.NewValidation()
	validation.Description = "XCTest: " + summary.Title
	validation.PassedCount = summary.PassedTests
	for _, item := range summary.TestFailures {
		path := item.TestIdentifier
		if path == "" {
			path = strings.TrimSpace(item.TargetName + "." + item.TestName)
		}
		failure := assertly.NewFailure("ios:test", path, assertly.EqualViolation, "passed", "failed")
		if item.FailureText != "" {
			failure.Message += ": " + item.FailureText
		}
		validation.AddFailure(failure)
	}
	missingFailures := summary.FailedTests - len(summary.TestFailures)
	for i := 0; i < missingFailures; i++ {
		validation.AddFailure(assertly.NewFailure("ios:test", fmt.Sprintf("failure[%d]", i), assertly.EqualViolation, "passed", "failed"))
	}
	return validation
}
