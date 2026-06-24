package goapitomcp

import (
	"encoding/json"
	"slices"

	"github.com/getkin/kin-openapi/openapi3"
)

func buildInputSchema(params []Parameter, body *RequestBody) map[string]any {
	schema := map[string]any{
		"type":                 "object",
		"additionalProperties": false,
	}

	properties := make(map[string]any)
	var required []string

	for _, group := range []string{
		openapi3.ParameterInPath,
		openapi3.ParameterInQuery,
		openapi3.ParameterInHeader,
		openapi3.ParameterInCookie,
	} {
		groupSchema, groupRequired, ok := buildParameterGroupSchema(params, group)
		if !ok {
			continue
		}
		properties[group] = groupSchema
		if len(groupRequired) > 0 {
			required = append(required, group)
		}
	}

	if body != nil {
		bodySchema := cloneSchemaMap(body.Schema)
		if len(bodySchema) == 0 {
			bodySchema = map[string]any{}
		}
		if body.Description != "" {
			if _, ok := bodySchema["description"]; !ok {
				bodySchema["description"] = body.Description
			}
		}
		properties["body"] = bodySchema
		if body.Required {
			required = append(required, "body")
		}
	}

	if len(properties) > 0 {
		schema["properties"] = properties
	}
	if len(required) > 0 {
		schema["required"] = required
	}
	return schema
}

func buildParameterGroupSchema(params []Parameter, group string) (map[string]any, []string, bool) {
	properties := make(map[string]any)
	var required []string

	for _, param := range params {
		if param.In != group {
			continue
		}
		paramSchema := cloneSchemaMap(param.Schema)
		if len(paramSchema) == 0 {
			paramSchema = map[string]any{}
		}
		if param.Description != "" {
			if _, ok := paramSchema["description"]; !ok {
				paramSchema["description"] = param.Description
			}
		}
		properties[param.Name] = paramSchema
		if param.Required {
			required = append(required, param.Name)
		}
	}
	if len(properties) == 0 {
		return nil, nil, false
	}
	slices.Sort(required)
	return map[string]any{
		"type":                 "object",
		"additionalProperties": false,
		"properties":           properties,
		"required":             required,
	}, required, true
}

func cloneSchemaMap(in map[string]any) map[string]any {
	if in == nil {
		return nil
	}
	data, err := json.Marshal(in)
	if err != nil {
		return nil
	}
	var out map[string]any
	if err := json.Unmarshal(data, &out); err != nil {
		return nil
	}
	return out
}
