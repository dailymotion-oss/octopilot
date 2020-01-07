package parameters

import (
	"strings"
)

// Parse parses a string representation of parameters, and return a key-value map.
// The string representation is a comma-separated list of key=value elements.
// For example: "key1=value1,key2=value2"
func Parse(paramsStr string) map[string]string {
	params := make(map[string]string)
	for _, param := range strings.Split(paramsStr, ",") {
		kv := strings.SplitN(param, "=", 2)
		if len(kv) == 2 {
			key := kv[0]
			value := kv[1]

			if strings.HasPrefix(value, `'`) && strings.HasSuffix(value, `'`) {
				value = value[1 : len(value)-1]
			}
			if strings.HasPrefix(value, `"`) && strings.HasSuffix(value, `"`) {
				value = value[1 : len(value)-1]
			}

			params[key] = value
		}
	}
	return params
}
