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
	xcodebuild, err := mobile.ResolveExecutable("xcodebuild", "/usr/bin/xcodebuild")
	if err != nil {
		return nil, err
	}
	buildCtx, cancel := context.WithTimeout(ctx.Background(), time.Duration(request.TimeoutMs)*time.Millisecond)
	defer cancel()
	response := &BuildResponse{Artifacts: []Artifact{}}
	if request.Mode != "export" {
		if request.Mode == "build" || request.Mode == "buildForTesting" {
			if err := s.validateLease(request.Destination); err != nil {
				return response, err
			}
		}
		args := iosBuildArgs(request)
		result, runErr := s.runner.Run(buildCtx, mobile.Command{Name: xcodebuild, Args: args})
		response.Stdout += result.Stdout
		response.Stderr += result.Stderr
		response.DurationMs += result.DurationMs
		if runErr != nil {
			return response, fmt.Errorf("Xcode %s: %w", request.Mode, runErr)
		}
		if request.Mode == "build" || request.Mode == "buildForTesting" {
			artifacts, err := discoverIOSBuildArtifacts(request.DerivedDataPath)
			if err != nil {
				return response, err
			}
			if len(artifacts) == 0 {
				return response, fmt.Errorf("Xcode build completed but no app or xctestrun products were found")
			}
			response.Artifacts = append(response.Artifacts, artifacts...)
		} else {
			artifact, err := iosProductArtifact("xcarchive", request.ArchivePath)
			if err != nil {
				return response, err
			}
			response.Artifacts = append(response.Artifacts, artifact)
		}
	}
	if request.Mode == "export" || request.Mode == "archiveAndExport" {
		result, runErr := s.runner.Run(buildCtx, mobile.Command{Name: xcodebuild, Args: []string{
			"-exportArchive", "-archivePath", request.ArchivePath,
			"-exportPath", request.ExportPath, "-exportOptionsPlist", request.ExportOptionsPlist,
		}})
		response.Stdout += result.Stdout
		response.Stderr += result.Stderr
		response.DurationMs += result.DurationMs
		if runErr != nil {
			return response, fmt.Errorf("Xcode export: %w", runErr)
		}
		artifacts, err := discoverExportedIPAs(request.ExportPath)
		if err != nil {
			return response, err
		}
		if len(artifacts) == 0 {
			return response, fmt.Errorf("Xcode export completed but no IPA was found")
		}
		response.Artifacts = append(response.Artifacts, artifacts...)
	}
	return response, nil
}

func iosBuildArgs(request *BuildRequest) []string {
	args := []string{}
	if request.WorkspacePath != "" {
		args = append(args, "-workspace", request.WorkspacePath)
	} else {
		args = append(args, "-project", request.ProjectPath)
	}
	args = append(args, "-scheme", request.Scheme, "-configuration", request.Configuration)
	if request.Mode == "build" || request.Mode == "buildForTesting" {
		destination := "platform=iOS Simulator,id=" + request.Destination.UDID
		if request.Destination.IsDevice() {
			destination = "platform=iOS,id=" + request.Destination.UDID
		}
		args = append(args, "-destination", destination, "-derivedDataPath", request.DerivedDataPath)
	} else {
		args = append(args, "-destination", "generic/platform=iOS", "-archivePath", request.ArchivePath)
		if request.SDK != "" {
			args = append(args, "-sdk", request.SDK)
		}
		if request.DerivedDataPath != "" {
			args = append(args, "-derivedDataPath", request.DerivedDataPath)
		}
	}
	settings := map[string]string{}
	for key, value := range request.BuildSettings {
		settings[key] = value
	}
	if request.Signing != nil {
		style := "Automatic"
		if strings.EqualFold(request.Signing.Style, "manual") {
			style = "Manual"
		}
		settings["CODE_SIGN_STYLE"] = style
		settings["DEVELOPMENT_TEAM"] = request.Signing.TeamID
		if request.Signing.Identity != "" {
			settings["CODE_SIGN_IDENTITY"] = request.Signing.Identity
		}
		if request.Signing.ProvisioningProfile != "" {
			settings["PROVISIONING_PROFILE_SPECIFIER"] = request.Signing.ProvisioningProfile
		}
		if request.Signing.KeychainPath != "" {
			settings["OTHER_CODE_SIGN_FLAGS"] = "--keychain " + request.Signing.KeychainPath
		}
	}
	keys := make([]string, 0, len(settings))
	for key := range settings {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		args = append(args, key+"="+settings[key])
	}
	switch request.Mode {
	case "buildForTesting":
		args = append(args, "build-for-testing")
	case "archive", "archiveAndExport":
		args = append(args, "archive")
	default:
		args = append(args, "build")
	}
	return args
}

func iosProductArtifact(kind, path string) (Artifact, error) {
	hash, size, err := hashBuildProduct(path)
	if err != nil {
		return Artifact{}, err
	}
	return Artifact{Kind: kind, HostPath: path, SHA256: hash, Size: size}, nil
}

func discoverExportedIPAs(exportPath string) ([]Artifact, error) {
	result := []Artifact{}
	err := filepath.WalkDir(exportPath, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() || !strings.HasSuffix(strings.ToLower(path), ".ipa") {
			return nil
		}
		artifact, err := iosProductArtifact("ipa", path)
		if err != nil {
			return err
		}
		result = append(result, artifact)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("discover exported IPAs: %w", err)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].HostPath < result[j].HostPath })
	return result, nil
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
