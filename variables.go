package linux_installer

import (
	"bytes"
	"fmt"
	"log"
	"strings"
	"text/template"
)

type StringMap map[string]string

// ExpandVariables takes a string with template variables like {{.var}} and expands them
// with the given map.
func ExpandVariables(str string, variables StringMap) (expanded string) {
	functions := template.FuncMap{
		"replace": func(from, to, input string) string { return strings.Replace(input, from, to, -1) },
		"trim":    func(input string) string { return strings.Trim(input, " \r\n\t") },
		"split":   func(sep, input string) []string { return strings.Split(input, sep) },
		"join":    func(sep string, input []string) string { return strings.Join(input, sep) },
		"upper":   func(input string) string { return strings.ToUpper(input) },
		"lower":   func(input string) string { return strings.ToLower(input) },
		"title":   func(input string) string { return strings.ToTitle(input) },
	}
	templ, err := template.New("").Funcs(functions).Parse(str)
	if err != nil {
		log.Println(fmt.Sprintf("Invalid string template: '%s'", err))
		return str
	}
	var buf bytes.Buffer
	err = templ.Execute(&buf, variables)
	if err != nil {
		log.Println(fmt.Sprintf("Error executing template: '%s'", err))
		return str
	}
	return buf.String()
}

// MergeVariables combines several variable maps into a single one. Duplicate keys will
// be overridden by the value in the last map which has the key.
func MergeVariables(varMaps ...StringMap) StringMap {
	merged := make(StringMap)
	for _, vars := range varMaps {
		for k, v := range vars {
			merged[k] = v
		}
	}
	return merged
}
