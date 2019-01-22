package gc

import (
	"fmt"
	"io"
	"net/http"
	"strings"
)



func Upload(httpClient *http.Client, signedURL string, reader io.Reader) error {
	request, err:= http.NewRequest("POST", signedURL, strings.NewReader(""))
	if err != nil {
		return err
	}
	request.Header.Set("x-goog-content-type", "application/zip")
	request.Header.Set("content-type", "application/zip")
	request.Header.Set("x-goog-resumable", "start")
	response, err := httpClient.Do(request)
	if err != nil {
		return err
	}
	if response.StatusCode != http.StatusCreated {
		return fmt.Errorf("failed to init upload code: %v",  response.Status)
	}

	 putRequest, err := http.NewRequest("PUT", signedURL, reader)
	 if err != nil {
	 	return err
	 }


	putRequest.Header.Set("content-type", "application/zip")
	putRequest.Header.Set("x-goog-content-type", "application/zip")
	putRequest.Header.Set("x-goog-content-length-range", "0,104857600")
	response, err = httpClient.Do(putRequest)
	 if err != nil {
	 	return err
	 }
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to init upload code: %v", response.Status)
	}
	return nil
}