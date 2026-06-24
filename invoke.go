package goapitomcp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const maxResponseBytes = 10 << 20

type toolPayload struct {
	Status      int    `json:"status"`
	ContentType string `json:"contentType,omitempty"`
	Body        any    `json:"body"`
}

func (c *Catalog) callOperation(ctx context.Context, op *Operation, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args, err := decodeArguments(req)
	if err != nil {
		return toolError(0, "text/plain", err.Error()), nil
	}

	httpReq, err := c.buildHTTPRequest(ctx, op, args)
	if err != nil {
		return toolError(0, "text/plain", err.Error()), nil
	}

	if headers, _ := ctx.Value(forwardedHeadersContextKey{}).(http.Header); len(headers) > 0 {
		applyHeaders(httpReq, headers)
	}

	if c.cfg.BeforeRequest != nil {
		if err := c.cfg.BeforeRequest(ctx, OperationContext{
			ToolName:    op.ToolName,
			OperationID: op.OperationID,
			Method:      op.Method,
			Path:        op.Path,
		}, httpReq); err != nil {
			return toolError(0, "text/plain", err.Error()), nil
		}
	}

	client := c.cfg.HTTPClient
	if client == nil {
		client = http.DefaultClient
	}
	resp, err := client.Do(httpReq)
	if err != nil {
		return toolError(0, "text/plain", err.Error()), nil
	}
	defer resp.Body.Close()

	body, readErr := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes+1))
	if readErr != nil {
		return toolError(resp.StatusCode, "text/plain", readErr.Error()), nil
	}
	if len(body) > maxResponseBytes {
		return toolError(resp.StatusCode, "text/plain", "response body exceeds 10 MiB limit"), nil
	}

	contentType := resp.Header.Get("Content-Type")
	parsedBody := parseResponseBody(contentType, body)
	payload := toolPayload{
		Status:      resp.StatusCode,
		ContentType: contentType,
		Body:        parsedBody,
	}
	return toolResult(payload, resp.StatusCode < 200 || resp.StatusCode >= 300), nil
}

func decodeArguments(req *mcp.CallToolRequest) (map[string]any, error) {
	if req == nil || req.Params == nil || len(req.Params.Arguments) == 0 {
		return map[string]any{}, nil
	}
	var args map[string]any
	if err := json.Unmarshal(req.Params.Arguments, &args); err != nil {
		return nil, fmt.Errorf("decode arguments: %w", err)
	}
	if args == nil {
		args = map[string]any{}
	}
	return args, nil
}

func (c *Catalog) buildHTTPRequest(ctx context.Context, op *Operation, args map[string]any) (*http.Request, error) {
	if c.BaseURL == nil {
		return nil, errorsNoBaseURL(c.specServers)
	}

	u := cloneURL(c.BaseURL)
	rawPath, err := c.operationRawPath(op, args)
	if err != nil {
		return nil, err
	}
	setJoinedPath(u, rawPath)

	query := u.Query()
	headers := make(http.Header)
	var cookies []*http.Cookie

	for _, param := range op.Parameters {
		group, ok, err := argumentGroup(args, param.In)
		if err != nil {
			return nil, err
		}
		if !ok {
			if param.Required {
				return nil, fmt.Errorf("missing required %s parameter %q", param.In, param.Name)
			}
			continue
		}
		value, ok := group[param.Name]
		if !ok || value == nil {
			if param.Required {
				return nil, fmt.Errorf("missing required %s parameter %q", param.In, param.Name)
			}
			continue
		}

		switch param.In {
		case openapi3.ParameterInQuery:
			addQueryParam(query, param, value)
		case openapi3.ParameterInHeader:
			headers.Set(param.Name, serializeParameterValue(param, value))
		case openapi3.ParameterInCookie:
			cookies = append(cookies, &http.Cookie{Name: param.Name, Value: serializeParameterValue(param, value)})
		}
	}
	u.RawQuery = query.Encode()

	body, contentType, err := requestBody(op, args)
	if err != nil {
		return nil, err
	}
	httpReq, err := http.NewRequestWithContext(ctx, op.Method, u.String(), body)
	if err != nil {
		return nil, err
	}
	for name, values := range headers {
		for _, value := range values {
			httpReq.Header.Add(name, value)
		}
	}
	for _, cookie := range cookies {
		httpReq.AddCookie(cookie)
	}
	if contentType != "" {
		httpReq.Header.Set("Content-Type", contentType)
	}
	httpReq.Header.Set("Accept", "application/json, */*")
	applyHeaders(httpReq, c.cfg.StaticHeaders)
	c.applySecurityDefaults(httpReq, op)
	return httpReq, nil
}

func errorsNoBaseURL(specServers []string) error {
	if len(specServers) == 0 {
		return fmt.Errorf("no BaseURL configured and the OpenAPI spec has no absolute server URL")
	}
	return fmt.Errorf("no BaseURL configured and no absolute OpenAPI server URL is usable: %s", strings.Join(specServers, ", "))
}

func (c *Catalog) operationRawPath(op *Operation, args map[string]any) (string, error) {
	pathArgs, _, err := argumentGroup(args, openapi3.ParameterInPath)
	if err != nil {
		return "", err
	}
	rawPath := op.Path
	for _, param := range op.Parameters {
		if param.In != openapi3.ParameterInPath {
			continue
		}
		value, ok := pathArgs[param.Name]
		if !ok || value == nil {
			return "", fmt.Errorf("missing required path parameter %q", param.Name)
		}
		rawPath = strings.ReplaceAll(rawPath, "{"+param.Name+"}", url.PathEscape(scalarToString(value)))
	}
	if strings.Contains(rawPath, "{") || strings.Contains(rawPath, "}") {
		return "", fmt.Errorf("path %q still contains unresolved template variables", op.Path)
	}
	return rawPath, nil
}

func setJoinedPath(u *url.URL, operationRawPath string) {
	basePath := u.EscapedPath()
	if basePath == "/" {
		basePath = ""
	}
	rawPath := strings.TrimRight(basePath, "/") + "/" + strings.TrimLeft(operationRawPath, "/")
	if rawPath == "" {
		rawPath = "/"
	}
	if decoded, err := url.PathUnescape(rawPath); err == nil {
		u.Path = decoded
		u.RawPath = rawPath
		return
	}
	u.Path = rawPath
	u.RawPath = ""
}

func argumentGroup(args map[string]any, name string) (map[string]any, bool, error) {
	raw, ok := args[name]
	if !ok || raw == nil {
		return map[string]any{}, false, nil
	}
	group, ok := raw.(map[string]any)
	if !ok {
		return nil, false, fmt.Errorf("argument group %q must be an object", name)
	}
	return group, true, nil
}

func requestBody(op *Operation, args map[string]any) (io.Reader, string, error) {
	if op.RequestBody == nil {
		return nil, "", nil
	}
	raw, hasBody := args["body"]
	if !hasBody || raw == nil {
		if op.RequestBody.Required {
			return nil, "", fmt.Errorf("missing required request body")
		}
		return nil, "", nil
	}
	if op.RequestBody.MediaType == "" {
		return nil, "", fmt.Errorf("operation %s %s does not have a supported JSON or form request body media type", op.Method, op.Path)
	}
	if op.RequestBody.MediaType == "application/x-www-form-urlencoded" {
		data, err := encodeFormBody(raw)
		if err != nil {
			return nil, "", err
		}
		return strings.NewReader(data.Encode()), op.RequestBody.MediaType, nil
	}
	data, err := json.Marshal(raw)
	if err != nil {
		return nil, "", fmt.Errorf("encode JSON request body: %w", err)
	}
	return bytes.NewReader(data), op.RequestBody.MediaType, nil
}

func applyHeaders(req *http.Request, headers http.Header) {
	for name, values := range headers {
		req.Header.Del(name)
		for _, value := range values {
			req.Header.Add(name, value)
		}
	}
}

func encodeFormBody(raw any) (url.Values, error) {
	object, ok := raw.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("form request body must be an object")
	}
	values := make(url.Values)
	keys := sortedMapKeys(object)
	for _, key := range keys {
		value := object[key]
		switch v := value.(type) {
		case []any:
			for _, item := range v {
				values.Add(key, scalarToString(item))
			}
		default:
			values.Set(key, scalarToString(value))
		}
	}
	return values, nil
}

func addQueryParam(values url.Values, param Parameter, value any) {
	switch v := value.(type) {
	case []any:
		if param.Style == "spaceDelimited" {
			values.Set(param.Name, joinValues(v, " "))
			return
		}
		if param.Style == "pipeDelimited" {
			values.Set(param.Name, joinValues(v, "|"))
			return
		}
		if param.Style == "form" && param.Explode {
			for _, item := range v {
				values.Add(param.Name, scalarToString(item))
			}
			return
		}
		values.Set(param.Name, joinValues(v, ","))
	case map[string]any:
		keys := sortedMapKeys(v)
		if param.Style == "form" && param.Explode {
			for _, key := range keys {
				values.Add(key, scalarToString(v[key]))
			}
			return
		}
		parts := make([]any, 0, len(keys)*2)
		for _, key := range keys {
			parts = append(parts, key, v[key])
		}
		values.Set(param.Name, joinValues(parts, ","))
	default:
		values.Set(param.Name, scalarToString(value))
	}
}

func serializeParameterValue(param Parameter, value any) string {
	switch v := value.(type) {
	case []any:
		return joinValues(v, ",")
	case map[string]any:
		keys := sortedMapKeys(v)
		parts := make([]any, 0, len(keys)*2)
		for _, key := range keys {
			parts = append(parts, key, v[key])
		}
		return joinValues(parts, ",")
	default:
		return scalarToString(value)
	}
}

func joinValues(values []any, separator string) string {
	parts := make([]string, 0, len(values))
	for _, value := range values {
		parts = append(parts, scalarToString(value))
	}
	return strings.Join(parts, separator)
}

func sortedMapKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	return keys
}

func scalarToString(value any) string {
	switch v := value.(type) {
	case string:
		return v
	case json.Number:
		return v.String()
	case bool:
		return strconv.FormatBool(v)
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64)
	case float32:
		return strconv.FormatFloat(float64(v), 'f', -1, 32)
	default:
		return fmt.Sprint(v)
	}
}

func parseResponseBody(contentType string, data []byte) any {
	if len(data) == 0 {
		return nil
	}
	if isJSONContentType(contentType) {
		var body any
		if err := json.Unmarshal(data, &body); err == nil {
			return body
		}
	}
	return string(data)
}

func isJSONContentType(contentType string) bool {
	mediaType, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		mediaType = contentType
	}
	return mediaType == "application/json" ||
		(strings.HasPrefix(mediaType, "application/") && strings.HasSuffix(mediaType, "+json"))
}

func toolError(status int, contentType string, message string) *mcp.CallToolResult {
	return toolResult(toolPayload{Status: status, ContentType: contentType, Body: message}, true)
}

func toolResult(payload toolPayload, isError bool) *mcp.CallToolResult {
	data, err := json.Marshal(payload)
	text := string(data)
	if err != nil {
		text = fmt.Sprintf(`{"status":0,"contentType":"text/plain","body":%q}`, err.Error())
	}
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: text},
		},
		StructuredContent: payload,
		IsError:           isError,
	}
}
