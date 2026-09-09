package ios

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/viant/endly"
)

func validateWDACompatibility(options *WDAOptions, destination DestinationLease) error {
	if options == nil {
		return nil
	}
	options.Init()
	if options.Mode != "preinstalled" {
		return nil
	}
	major := iosRuntimeMajor(destination.Runtime)
	if major > 0 && major < 17 {
		return fmt.Errorf("preinstalled WDA requires iOS 17 or newer; use managed, prebuilt, or external WDA")
	}
	if major >= 27 {
		if destination.IsDevice() {
			return fmt.Errorf("preinstalled WDA on iOS 27+ physical devices requires a working RemoteXPC tunnel, which this local runner does not configure; use managed, prebuilt, or external WDA")
		}
		return fmt.Errorf("preinstalled WDA is unavailable for iOS 27+ Simulator runtimes because direct XCTest runner launch cannot remain active; use managed, prebuilt, or external WDA")
	}
	return nil
}

func iosRuntimeMajor(runtime string) int {
	for _, marker := range []string{"iOS-", "iOS "} {
		if index := strings.Index(runtime, marker); index >= 0 {
			value := runtime[index+len(marker):]
			end := 0
			for end < len(value) && value[end] >= '0' && value[end] <= '9' {
				end++
			}
			if end > 0 {
				major, _ := strconv.Atoi(value[:end])
				return major
			}
		}
	}
	return 0
}

func wdaCapabilities(ctx *endly.Context, options *WDAOptions) (map[string]interface{}, error) {
	if options == nil {
		options = &WDAOptions{}
	}
	options.Init()
	if err := options.Validate(); err != nil {
		return nil, err
	}
	result := map[string]interface{}{}
	derivedDataPath := ctx.Expand(options.DerivedDataPath)
	prebuiltWDAPath := ctx.Expand(options.PrebuiltWDAPath)
	webDriverAgentURL := ctx.Expand(options.WebDriverAgentURL)
	switch options.Mode {
	case "managed":
	case "prebuilt":
		if err := requireDirectory(derivedDataPath, "prebuilt WDA DerivedDataPath"); err != nil {
			return nil, err
		}
		result["appium:usePrebuiltWDA"] = true
	case "preinstalled":
		result["appium:usePreinstalledWDA"] = true
		if prebuiltWDAPath != "" {
			if err := requireDirectory(prebuiltWDAPath, "preinstalled WDA application"); err != nil {
				return nil, err
			}
			if !strings.HasSuffix(strings.ToLower(prebuiltWDAPath), ".app") {
				return nil, fmt.Errorf("PrebuiltWDAPath must identify an .app bundle")
			}
			result["appium:prebuiltWDAPath"] = prebuiltWDAPath
		}
	case "external":
		result["appium:webDriverAgentUrl"] = webDriverAgentURL
	}
	if derivedDataPath != "" {
		result["appium:derivedDataPath"] = derivedDataPath
	}
	if options.UpdatedBundleID != "" {
		result["appium:updatedWDABundleId"] = options.UpdatedBundleID
	}
	if options.UpdatedBundleIDSuffix != nil {
		result["appium:updatedWDABundleIdSuffix"] = *options.UpdatedBundleIDSuffix
	}
	if options.XcodeOrgID != "" {
		result["appium:xcodeOrgId"] = options.XcodeOrgID
		result["appium:xcodeSigningId"] = options.XcodeSigningID
	}
	if options.XcodeConfigFile != "" {
		path := ctx.Expand(options.XcodeConfigFile)
		if err := requireFile(path, "WDA XcodeConfigFile"); err != nil {
			return nil, err
		}
		result["appium:xcodeConfigFile"] = path
	}
	if options.KeychainPath != "" {
		keychainPath := ctx.Expand(options.KeychainPath)
		if err := requireFile(keychainPath, "WDA KeychainPath"); err != nil {
			return nil, err
		}
		password, err := readSecretFile(ctx.Expand(options.KeychainPasswordFile))
		if err != nil {
			return nil, fmt.Errorf("read WDA keychain password: %w", err)
		}
		result["appium:keychainPath"] = keychainPath
		result["appium:keychainPassword"] = password
	}
	if options.LocalPort > 0 {
		result["appium:wdaLocalPort"] = options.LocalPort
	}
	if options.MJPEGServerPort > 0 {
		result["appium:mjpegServerPort"] = options.MJPEGServerPort
	}
	if options.UseNewWDA != nil {
		result["appium:useNewWDA"] = *options.UseNewWDA
	}
	if options.PrebuildWDA {
		result["appium:prebuildWDA"] = true
	}
	if options.LaunchTimeoutMs > 0 {
		result["appium:wdaLaunchTimeout"] = options.LaunchTimeoutMs
	}
	if options.ConnectionTimeoutMs > 0 {
		result["appium:wdaConnectionTimeout"] = options.ConnectionTimeoutMs
	}
	if options.StartupRetries > 0 {
		result["appium:wdaStartupRetries"] = options.StartupRetries
	}
	if options.StartupRetryMs > 0 {
		result["appium:wdaStartupRetryInterval"] = options.StartupRetryMs
	}
	return result, nil
}

func requireDirectory(path, label string) error {
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("%s %q: %w", label, path, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("%s %q is not a directory", label, path)
	}
	return nil
}

func requireFile(path, label string) error {
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("%s %q: %w", label, path, err)
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("%s %q is not a regular file", label, path)
	}
	return nil
}

func readSecretFile(path string) (string, error) {
	info, err := os.Stat(path)
	if err != nil {
		return "", err
	}
	if !info.Mode().IsRegular() || info.Size() > 64<<10 {
		return "", fmt.Errorf("secret must be a regular file no larger than 64 KiB")
	}
	if info.Mode().Perm()&0o077 != 0 {
		return "", fmt.Errorf("secret file %q must not be group/world accessible", filepath.Clean(path))
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	password := strings.TrimSpace(string(data))
	if password == "" {
		return "", fmt.Errorf("secret file %q was empty", filepath.Clean(path))
	}
	return password, nil
}
