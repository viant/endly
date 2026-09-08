package android

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
	wrapper := filepath.Join(request.ProjectDir, "gradlew")
	if _, err := mobile.ResolveExecutable(wrapper); err != nil {
		return nil, fmt.Errorf("resolve Gradle wrapper: %w", err)
	}
	args := []string{"--console=plain"}
	args = append(args, request.Tasks...)
	args = append(args, request.GradleArgs...)
	buildCtx, cancel := context.WithTimeout(ctx.Background(), time.Duration(request.TimeoutMs)*time.Millisecond)
	defer cancel()
	result, err := s.runner.Run(buildCtx, mobile.Command{Name: wrapper, Args: args, Dir: request.ProjectDir})
	response := &BuildResponse{Stdout: result.Stdout, Stderr: result.Stderr, DurationMs: result.DurationMs}
	if err != nil {
		return response, fmt.Errorf("Gradle build: %w", err)
	}
	artifacts, err := discoverAndroidArtifacts(request.ProjectDir, request.Module, request.Variant)
	if err != nil {
		return response, err
	}
	response.Artifacts = artifacts
	for _, task := range request.Tasks {
		if strings.Contains(strings.ToLower(task), "assemble") && len(artifacts) == 0 {
			return response, fmt.Errorf("Gradle assemble completed but no %s artifacts were found", request.Variant)
		}
	}
	return response, nil
}

func discoverAndroidArtifacts(projectDir, module, variant string) ([]BuildArtifact, error) {
	root := filepath.Join(projectDir, filepath.FromSlash(strings.Trim(module, ":/")), "build", "outputs")
	result := []BuildArtifact{}
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			if os.IsNotExist(err) {
				return nil
			}
			return err
		}
		if entry.IsDir() {
			return nil
		}
		extension := strings.ToLower(filepath.Ext(path))
		if extension != ".apk" && extension != ".aab" && extension != ".apks" {
			return nil
		}
		if variant != "" && !strings.Contains(strings.ToLower(path), strings.ToLower(variant)) {
			return nil
		}
		artifact, err := androidBuildArtifact(path)
		if err != nil {
			return err
		}
		result = append(result, artifact)
		return nil
	})
	if err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("discover Android artifacts: %w", err)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Path < result[j].Path })
	return result, nil
}

func androidBuildArtifact(path string) (BuildArtifact, error) {
	file, err := os.Open(path)
	if err != nil {
		return BuildArtifact{}, err
	}
	defer file.Close()
	hash := sha256.New()
	size, err := io.Copy(hash, file)
	if err != nil {
		return BuildArtifact{}, err
	}
	kind := strings.TrimPrefix(strings.ToLower(filepath.Ext(path)), ".")
	if kind == "apk" {
		kind = "appAPK"
		lower := strings.ToLower(path)
		if strings.Contains(lower, "androidtest") || strings.Contains(lower, "-test") {
			kind = "testAPK"
		}
	}
	return BuildArtifact{Kind: kind, Path: path, SHA256: fmt.Sprintf("%x", hash.Sum(nil)), Size: size}, nil
}
