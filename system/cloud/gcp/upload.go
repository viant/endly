package gcp

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
)

func Upload(httpClient *http.Client, uploadURL string, reader io.Reader) error {
	data, err := ioutil.ReadAll(reader)
	if err != nil {
		return err
	}
	request, err := http.NewRequest("PUT", uploadURL, bytes.NewReader(data))
	if err != nil {
		return err
	}
	request.Header.Set("content-type", "application/zip")
	request.Header.Set("x-goog-content-length-range", "0,104857600")
	request.Header.Set("Content-Length", fmt.Sprintf("%d", len(data)))
	response, err := httpClient.Do(request)
	if err != nil {
		return err
	}
	var message []byte
	if response.ContentLength > 0 {
		message, err = ioutil.ReadAll(response.Body)
	}
	if response.StatusCode/100 != 2 {
		return fmt.Errorf("failed to upload code: %v, %s", response.Status, message)
	}
	return nil
}
