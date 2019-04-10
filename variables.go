package linux_installer

import (
	"bytes"
	"fmt"
	"log"
	"strings"
	"text/template"
)

type (
	// VariableMap is string-to-string lookup.
	VariableMap map[string]string
	// UntypedVariableMap is a string-to-interface{} lookup
	UntypedVariableMap map[string]interface{}
)

var (
	varTemplateFunctions = template.FuncMap{
		"replace": func(from, to, input string) string { return strings.Replace(input, from, to, -1) },
		"trim":    func(input string) string { return strings.TrimSpace(input) },
		"split":   func(sep, input string) []string { return strings.Split(input, sep) },
		"join":    func(sep string, input []string) string { return strings.Join(input, sep) },
		"upper":   func(input string) string { return strings.ToUpper(input) },
		"lower":   func(input string) string { return strings.ToLower(input) },
		"title":   func(input string) string { return strings.ToTitle(input) },
	}
)

// ExpandVariables takes a string with template variables like {{.var}} and expands them
// with the given variables map.
func ExpandVariables(str string, variables VariableMap) (expanded string) {
	return ExpandAllVariables(str, variables, map[string]interface{}{})
}

// ExpandAllVariables is the same as ExpandVariables, except that it additionally takes
// untypedVariables, a string map of values of arbitrary type.
func ExpandAllVariables(
	str string,
	variables VariableMap,
	untypedVariables map[string]interface{},
) (expanded string) {
	return expandAllVariablesRecursively(str, variables, untypedVariables, 0)
}

// expandAllVariablesRecursively expands variables, possibly recursively if the expanded
// value contains template variables as well.
func expandAllVariablesRecursively(
	str string,
	variables VariableMap,
	untypedVariables map[string]interface{},
	depth int,
) (expanded string) {
	for k, v := range variables {
		untypedVariables[k] = v
	}
	templ, err := template.New("").Funcs(varTemplateFunctions).Parse(str)
	if err != nil {
		log.Println(fmt.Sprintf("Invalid string template: '%s'", err))
		return str
	}
	var buf bytes.Buffer
	err = templ.Execute(&buf, untypedVariables)
	if err != nil {
		log.Println(fmt.Sprintf("Error executing template: '%s'", err))
		return str
	}
	expanded = buf.String()
	if depth <= 3 && strings.Contains(expanded, "{{") {
		expanded = expandAllVariablesRecursively(
			expanded, variables, untypedVariables, depth+1,
		)
	}
	return
}

// MergeVariables combines several variable maps into a single one. Duplicate keys will
// be overridden by the value in the last map which has the key.
func MergeVariables(varMaps ...VariableMap) VariableMap {
	merged := make(VariableMap)
	for _, vars := range varMaps {
		for k, v := range vars {
			merged[k] = v
		}
	}
	return merged
}
