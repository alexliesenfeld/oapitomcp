package goapitomcp

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"slices"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
)

func collectSecuritySchemes(doc *openapi3.T) []SecurityScheme {
	if doc == nil || doc.Components == nil || len(doc.Components.SecuritySchemes) == 0 {
		return nil
	}
	ids := make([]string, 0, len(doc.Components.SecuritySchemes))
	for id := range doc.Components.SecuritySchemes {
		ids = append(ids, id)
	}
	slices.Sort(ids)

	schemes := make([]SecurityScheme, 0, len(ids))
	for _, id := range ids {
		ref := doc.Components.SecuritySchemes[id]
		if ref == nil || ref.Value == nil {
			continue
		}
		scheme := ref.Value
		schemes = append(schemes, SecurityScheme{
			ID:                id,
			Type:              scheme.Type,
			Scheme:            scheme.Scheme,
			In:                scheme.In,
			Name:              scheme.Name,
			Description:       scheme.Description,
			BearerFormat:      scheme.BearerFormat,
			OpenIDConnectURL:  scheme.OpenIdConnectUrl,
			DefaultCredential: defaultCredential(scheme),
		})
	}
	return schemes
}

func defaultCredential(scheme *openapi3.SecurityScheme) string {
	if scheme == nil {
		return ""
	}
	if value, ok := scheme.Extensions["x-defaultCredential"]; ok {
		return extensionString(value)
	}
	if value, ok := scheme.Extensions["defaultCredential"]; ok {
		return extensionString(value)
	}
	return ""
}

func extensionString(value any) string {
	if value == nil {
		return ""
	}
	if s, ok := value.(string); ok {
		return s
	}
	return fmt.Sprint(value)
}

func effectiveSecurity(operation *openapi3.Operation, root openapi3.SecurityRequirements) []SecurityRequirement {
	var requirements openapi3.SecurityRequirements
	if operation.Security != nil {
		requirements = *operation.Security
	} else {
		requirements = root
	}
	if len(requirements) == 0 {
		return nil
	}
	out := make([]SecurityRequirement, 0)
	for _, requirement := range requirements {
		ids := make([]string, 0, len(requirement))
		for id := range requirement {
			ids = append(ids, id)
		}
		slices.Sort(ids)
		for _, id := range ids {
			scopes := append([]string(nil), requirement[id]...)
			slices.Sort(scopes)
			out = append(out, SecurityRequirement{ID: id, Scopes: scopes})
		}
	}
	return out
}

func securityRequirementNames(requirements []SecurityRequirement) []string {
	names := make([]string, 0, len(requirements))
	seen := make(map[string]struct{}, len(requirements))
	for _, requirement := range requirements {
		if _, ok := seen[requirement.ID]; ok {
			continue
		}
		seen[requirement.ID] = struct{}{}
		names = append(names, requirement.ID)
	}
	slices.Sort(names)
	return names
}

func (c *Catalog) applySecurityDefaults(req *http.Request, op *Operation) {
	if !c.cfg.UseSecurityDefaults || len(op.Security) == 0 || len(c.SecuritySchemes) == 0 {
		return
	}
	schemes := make(map[string]SecurityScheme, len(c.SecuritySchemes))
	for _, scheme := range c.SecuritySchemes {
		schemes[scheme.ID] = scheme
	}
	for _, requirement := range op.Security {
		scheme, ok := schemes[requirement.ID]
		if !ok || scheme.DefaultCredential == "" {
			continue
		}
		applySecurityCredential(req, scheme)
	}
}

func applySecurityCredential(req *http.Request, scheme SecurityScheme) {
	switch scheme.Type {
	case "apiKey":
		applyAPIKeyCredential(req, scheme)
	case "http":
		applyHTTPSecurityCredential(req, scheme)
	case "oauth2", "openIdConnect":
		setAuthorization(req, "Bearer "+scheme.DefaultCredential)
	}
}

func applyAPIKeyCredential(req *http.Request, scheme SecurityScheme) {
	if scheme.Name == "" {
		return
	}
	switch scheme.In {
	case "query":
		q := req.URL.Query()
		if q.Get(scheme.Name) == "" {
			q.Set(scheme.Name, scheme.DefaultCredential)
			req.URL.RawQuery = q.Encode()
		}
	case "cookie":
		if _, err := req.Cookie(scheme.Name); err == nil {
			return
		}
		req.AddCookie(&http.Cookie{Name: scheme.Name, Value: scheme.DefaultCredential})
	case "header", "":
		if req.Header.Get(scheme.Name) == "" {
			req.Header.Set(scheme.Name, scheme.DefaultCredential)
		}
	}
}

func applyHTTPSecurityCredential(req *http.Request, scheme SecurityScheme) {
	switch strings.ToLower(scheme.Scheme) {
	case "basic":
		credential := scheme.DefaultCredential
		if strings.Contains(credential, ":") {
			credential = base64.StdEncoding.EncodeToString([]byte(credential))
		}
		setAuthorization(req, "Basic "+credential)
	case "bearer":
		setAuthorization(req, "Bearer "+scheme.DefaultCredential)
	default:
		setAuthorization(req, scheme.DefaultCredential)
	}
}

func setAuthorization(req *http.Request, value string) {
	if req.Header.Get("Authorization") == "" {
		req.Header.Set("Authorization", value)
	}
}
