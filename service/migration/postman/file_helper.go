package postman

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/viant/toolbox"
)

const JSONExt = ".json"

//*** IO Functions - harder to unit test ***

func getPostmanObjects(path string) ([]*postmanObject, error) {
	var objects []*postmanObject

	jsonFilePaths, err := getJSONFilePaths(path)
	if err != nil {
		return nil, err
	}

	for _, p := range jsonFilePaths {
		o, err := parsePostmanFile(p)
		if err != nil {
			return nil, err
		}
		objects = append(objects, o)
	}

	return objects, nil
}

func parsePostmanFile(path string) (*postmanObject, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	return parsePostmanReader(f)
}

func getJSONFilePaths(path string) ([]string, error) {
	var jsonFilePaths []string
	if toolbox.IsDirectory(path) {
		var err error
		jsonFilePaths, err = getJsonFilesFromDir(path)
		if err != nil {
			return nil, err
		}
	} else if toolbox.FileExists(path) {
		if hasJSONExtension(path) {
			jsonFilePaths = append(jsonFilePaths, path)
		} else {
			return nil, fmt.Errorf("file is not a *.json file")
		}
	} else {
		return nil, fmt.Errorf("invalid collection path")
	}

	return jsonFilePaths, nil
}

func getJsonFilesFromDir(path string) ([]string, error) {
	var jsonFiles []string
	err := filepath.Walk(path, func(filePath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if hasJSONExtension(info.Name()) {
			jsonFiles = append(jsonFiles, filePath)
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	return jsonFiles, nil
}

func hasJSONExtension(path string) bool {
	return strings.Contains(path, JSONExt)
}
