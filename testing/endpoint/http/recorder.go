package http

import (
	"fmt"
	"github.com/satori/go.uuid"
	"github.com/viant/toolbox"
	"github.com/viant/toolbox/bridge"
	"log"
	"net/url"
	"os"
	"path"
	"strings"
)

// StartRecorder starts HTTP recorded for supplied URLs
func StartRecorder(targetURLs ...string) error {
	if len(targetURLs) == 0 {
		return fmt.Errorf("target URLs were empty")
	}
	var targetURL = targetURLs[0]
	URL, err := url.Parse(targetURL)
	if err != nil {
		return err
	}
	port := URL.Port()
	isSecure := strings.HasPrefix(targetURL, "https:")

	UUID, err := uuid.NewV1()
	if err != nil {
		return err
	}
	currentDirectory, _ := os.Getwd()

	var outputDirectory = path.Join(currentDirectory, fmt.Sprintf("http_recording-%v", UUID.String()))
	log.Printf("capturing HTTP trafic to %v", outputDirectory)
	if port == "" {
		if isSecure {
			port = "443"
		} else {
			port = "80"
		}
	}

	var routes = []*bridge.HttpBridgeProxyRoute{}
	for _, targetURL := range targetURLs {
		URL, err := url.Parse(targetURL)
		if err != nil {
			return fmt.Errorf("failed to parse URL %v, %v", targetURL, err)
		}

		urlPath := URL.Path
		if urlPath == "" {
			urlPath = "/"
		}
		routes = append(routes,
			&bridge.HttpBridgeProxyRoute{
				Pattern:   urlPath,
				TargetURL: URL,
			})
	}
	recorderBridge, err := bridge.StartRecordingBridge(port, outputDirectory, routes...)
	if isSecure {
		var serverCert = "server.crt"
		var serverKey = "server.key"
		if toolbox.FileExists(serverCert) {
			return fmt.Errorf("SSL server cert file does not exists %v", serverCert)
		}
		if toolbox.FileExists(serverCert) {
			return fmt.Errorf("SSL server key file does not exists %v", serverKey)
		}
		return recorderBridge.ListenAndServeTLS(serverCert, serverKey)
	}
	return recorderBridge.ListenAndServe()
}
