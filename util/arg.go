package util

import (
	"fmt"
	"github.com/viant/toolbox"
	"github.com/viant/toolbox/data"
	"os"
	"strings"
)

//GetArguments extract cli arguments into a map
func GetArguments(baseURLs ...string) (map[string]interface{}, error) {
	var args = data.NewMap()
	var arguments = make([]interface{}, 0)
	if len(os.Args) > 1 {
		offset := 1
		if strings.Contains(os.Args[1], ":") || strings.Contains(os.Args[1], ".") && len(os.Args) > 2 {
			offset = 2
		}
		for i := offset ; i < len(os.Args); i++ {
			var candidate = os.Args[i]
			if strings.HasPrefix(candidate, "-") {
				if !strings.Contains(candidate, "=") {
					i++
				}
				continue
			}
			keyValuePair := strings.SplitN(candidate, "=", 2)
			if len(keyValuePair) == 2 {
				normalized, err := normalizeArgument(baseURLs, keyValuePair[1])
				if err != nil {
					return nil, err
				}
				arguments = append(arguments, keyValuePair[0], normalized)
			} else {
				normalized, err := normalizeArgument(baseURLs, candidate)
				if err != nil {
					return nil, err
				}
				arguments = append(arguments, normalized)
			}
			arguments = append(arguments)
		}
	}
	for i := 0; i < len(arguments); i += 2 {
		if i+1 >= len(arguments) {
			break
		}
		args.SetValue(toolbox.AsString(arguments[i]), arguments[i+1])
	}
	return args, nil
}

func normalizeArgument(baseURLs []string, value string) (interface{}, error) {
	value = strings.Trim(value, " \"'")
	if !(strings.HasPrefix(value, "#") || strings.HasPrefix(value, "@")) {
		return value, nil
	}

	loaded, err := LoadData(baseURLs, value)
	if err != nil {
		return nil, fmt.Errorf("failed to load: %v, %v", value, err)
	}
	if loaded != nil && (toolbox.IsSlice(loaded) || toolbox.IsMap(loaded)) {
		if kvNormalizer, err := toolbox.NormalizeKVPairs(loaded); err == nil {
			return kvNormalizer, nil
		}
	}
	return loaded, nil
}
