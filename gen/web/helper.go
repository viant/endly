package web

import (
	"github.com/viant/toolbox/storage"
	"io/ioutil"
	"strings"
)

func download(service storage.Service, URL string, result map[string]string) (err error) {
	objects, err := service.List(URL)
	if err != nil {
		return err
	}

	for _, object := range objects {
		objectURL := object.URL()
		if URL == objectURL {
			continue
		}
		if object.IsContent() {
			reader, err := service.Download(object)
			if err != nil {
				return err
			}
			defer reader.Close()
			content, err := ioutil.ReadAll(reader)
			result[object.URL()] = string(content)
		} else {
			if err = download(service, object.URL(), result); err != nil {
				return err
			}
		}
	}
	return nil
}



//DownloadAll downloads assets from supplied URL
func DownloadAll(URL string) (map[string]string, error) {
	service, err := storage.NewServiceForURL(URL, "")
	if err != nil {
		return nil, err
	}

	if strings.HasSuffix(URL, "/") {
		URL = string(URL[:len(URL)-1])
	}
	var temp = make(map[string]string)
	if err = download(service, URL, temp); err != nil {
		return nil, err
	}
	var result = make(map[string]string)
	URL += "/"
	for k, v := range temp {
		var key = strings.Replace(k, URL, "", 1)
		result[key] = v
	}
	return result, nil
}
