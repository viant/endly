package docker

import (
	"archive/tar"
	"bytes"
	"fmt"
	"github.com/viant/endly/model/location"
	"github.com/viant/toolbox"
	"github.com/viant/toolbox/storage"
	"io"
	"io/ioutil"
	"os"
	"path"
)

// AsTarReader creates a tar reader for supplied URL
func AsTarReader(resource *location.Resource, includeOwnerDir bool) (io.Reader, error) {
	storageService, err := storage.NewServiceForURL(resource.URL, resource.Credentials)
	if err != nil {
		return nil, err
	}
	writer := new(bytes.Buffer)
	archive := tar.NewWriter(writer)
	if err = storage.Tar(storageService, resource.URL, archive, includeOwnerDir); err != nil {
		return nil, err
	}
	err = archive.Close()
	return writer, err
}

// UnTar write archive content to dest
func UnTar(reader *tar.Reader, dest string) error {
	var dirs = make(map[string]bool)
	for {
		header, err := reader.Next()
		if err != nil {
			if io.EOF == err {
				break
			}
			return err
		}
		if header.Size == 0 {
			_ = toolbox.CreateDirIfNotExist(path.Join(dest, header.Name))
			continue
		}
		var data = make([]byte, header.Size)

		readBytes, err := reader.Read(data)
		if readBytes != int(header.Size) {
			return fmt.Errorf("failed to read: %v, %v", header.Name, err)
		}
		var filename string
		if !toolbox.IsDirectory(dest) {
			filename = dest
		} else {
			filename = path.Join(dest, header.Name)
		}
		parent, _ := path.Split(filename)
		if _, has := dirs[parent]; !has {
			dirs[parent] = true
			_ = toolbox.CreateDirIfNotExist(parent)
		}
		if err = ioutil.WriteFile(filename, data, os.FileMode(header.FileInfo().Mode())); err != nil {
			return err
		}
	}
	return nil
}
