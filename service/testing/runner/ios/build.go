package ios

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/viant/endly"
	"github.com/viant/endly/service/testing/runner/internal/mobile"
)

func (s *service) build(ctx *endly.Context, request *BuildRequest) (*BuildResponse, error) {
	if err := s.validateLease(request.Destination); err != nil {
		return nil, err
	}
	xcodebuild, err := mobile.ResolveExecutable("xcodebuild", "/usr/bin/xcodebuild")
	if err != nil {
		return nil, err
	}
	args := []string{}
	if request.WorkspacePath != "" {
		args = append(args, "-workspace", request.WorkspacePath)
	} else {
		args = append(args, "-project", request.ProjectPath)
	}
	args = append(args,
		"-scheme", request.Scheme,
		"-configuration", request.Configuration,
		"-destination", "platform=iOS Simulator,id="+request.Destination.UDID,
		"-derivedDataPath", request.DerivedDataPath,
	)
	keys := make([]string, 0, len(request.BuildSettings))
	for key := range request.BuildSettings {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		args = append(args, key+"="+request.BuildSettings[key])
	}
	if request.Mode == "buildForTesting" {
		args = append(args, "build-for-testing")
	} else {
		args = append(args, "build")
	}
	buildCtx, cancel := context.WithTimeout(ctx.Background(), time.Duration(request.TimeoutMs)*time.Millisecond)
	defer cancel()
	result, err := s.runner.Run(buildCtx, mobile.Command{Name: xcodebuild, Args: args})
	response := &BuildResponse{Stdout: result.Stdout, Stderr: result.Stderr, DurationMs: result.DurationMs}
	if err != nil {
		return response, fmt.Errorf("Xcode build: %w", err)
	}
	artifacts, err := discoverIOSBuildArtifacts(request.DerivedDataPath)
	if err != nil {
		return response, err
	}
	if len(artifacts) == 0 {
		return response, fmt.Errorf("Xcode build completed but no app or xctestrun products were found")
	}
	response.Artifacts = artifacts
	return response, nil
}

func discoverIOSBuildArtifacts(derivedDataPath string) ([]Artifact, error) {
	root := filepath.Join(derivedDataPath, "Build", "Products")
	result := []Artifact{}
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			if os.IsNotExist(err) {
				return nil
			}
			return err
		}
		lower := strings.ToLower(path)
		if entry.IsDir() && strings.HasSuffix(lower, ".app") {
			kind := "deviceApp"
			if strings.Contains(lower, "iphonesimulator") {
				kind = "simulatorApp"
			}
			hash, size, err := hashBuildProduct(path)
			if err != nil {
				return err
			}
			result = append(result, Artifact{Kind: kind, HostPath: path, SHA256: hash, Size: size})
			return filepath.SkipDir
		}
		if !entry.IsDir() && strings.HasSuffix(lower, ".xctestrun") {
			hash, size, err := hashBuildProduct(path)
			if err != nil {
				return err
			}
			result = append(result, Artifact{Kind: "xctestrun", HostPath: path, SHA256: hash, Size: size})
		}
		return nil
	})
	if err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("discover iOS build products: %w", err)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].HostPath < result[j].HostPath })
	return result, nil
}

func hashBuildProduct(productPath string) (string, int64, error) {
	info, err := os.Stat(productPath)
	if err != nil {
		return "", 0, err
	}
	hash := sha256.New()
	var size int64
	if !info.IsDir() {
		file, err := os.Open(productPath)
		if err != nil {
			return "", 0, err
		}
		defer file.Close()
		size, err = io.Copy(hash, file)
		return fmt.Sprintf("%x", hash.Sum(nil)), size, err
	}
	files := []string{}
	if err := filepath.WalkDir(productPath, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !entry.IsDir() {
			files = append(files, path)
		}
		return nil
	}); err != nil {
		return "", 0, err
	}
	sort.Strings(files)
	for _, path := range files {
		relative, _ := filepath.Rel(productPath, path)
		_, _ = io.WriteString(hash, filepath.ToSlash(relative)+"\x00")
		file, err := os.Open(path)
		if err != nil {
			return "", 0, err
		}
		written, copyErr := io.Copy(hash, file)
		closeErr := file.Close()
		size += written
		if copyErr != nil {
			return "", 0, copyErr
		}
		if closeErr != nil {
			return "", 0, closeErr
		}
	}
	return fmt.Sprintf("%x", hash.Sum(nil)), size, nil
}
