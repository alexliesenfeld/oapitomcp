package goapitomcp

import "strings"

// Allows reports whether an operation with the given metadata should be exposed
// as an MCP tool.
func (f *OperationFilter) Allows(method string, path string, operationID string, tags []string) bool {
	if f == nil {
		return true
	}
	return f.matchesIncludes(method, path, operationID, tags) &&
		!f.matchesExcludes(method, path, operationID, tags)
}

func (f *OperationFilter) matchesIncludes(method string, path string, operationID string, tags []string) bool {
	if hasPathFilters(f.IncludeOnlyPaths, f.IncludeOnlyPathPatterns) &&
		!matchesPathFilters(path, f.IncludeOnlyPaths, f.IncludeOnlyPathPatterns) {
		return false
	}
	if len(f.IncludeOnlyOperationIDs) > 0 && !containsString(f.IncludeOnlyOperationIDs, operationID) {
		return false
	}
	if len(f.IncludeOnlyMethods) > 0 && !containsFold(f.IncludeOnlyMethods, method) {
		return false
	}
	if len(f.IncludeOnlyTags) > 0 && !anyFold(tags, f.IncludeOnlyTags) {
		return false
	}
	return true
}

func (f *OperationFilter) matchesExcludes(method string, path string, operationID string, tags []string) bool {
	return matchesPathFilters(path, f.ExcludePaths, f.ExcludePathPatterns) ||
		(operationID != "" && containsString(f.ExcludeOperationIDs, operationID)) ||
		containsFold(f.ExcludeMethods, method) ||
		anyFold(tags, f.ExcludeTags)
}

func hasPathFilters(paths []string, patterns []string) bool {
	return len(paths) > 0 || len(patterns) > 0
}

func matchesPathFilters(path string, paths []string, patterns []string) bool {
	for _, candidate := range paths {
		if path == candidate {
			return true
		}
	}
	for _, pattern := range patterns {
		if matchPathPattern(pattern, path) {
			return true
		}
	}
	return false
}

func containsString(values []string, value string) bool {
	for _, candidate := range values {
		if candidate == value {
			return true
		}
	}
	return false
}

func containsFold(values []string, value string) bool {
	for _, candidate := range values {
		if strings.EqualFold(candidate, value) {
			return true
		}
	}
	return false
}

func anyFold(values []string, candidates []string) bool {
	for _, value := range values {
		if containsFold(candidates, value) {
			return true
		}
	}
	return false
}

func matchPathPattern(pattern string, path string) bool {
	if pattern == path {
		return true
	}
	patternSegments := splitPathPattern(pattern)
	pathSegments := splitPathPattern(path)
	if len(patternSegments) != len(pathSegments) {
		return false
	}
	for i, patternSegment := range patternSegments {
		if isTemplateSegment(patternSegment) {
			patternSegment = "*"
		}
		if !matchSegmentPattern(patternSegment, pathSegments[i]) {
			return false
		}
	}
	return true
}

func splitPathPattern(path string) []string {
	trimmed := strings.Trim(path, "/")
	if trimmed == "" {
		return []string{}
	}
	return strings.Split(trimmed, "/")
}

func isTemplateSegment(segment string) bool {
	return strings.HasPrefix(segment, "{") && strings.HasSuffix(segment, "}") && len(segment) > 2
}

func matchSegmentPattern(pattern string, value string) bool {
	if pattern == "*" {
		return true
	}
	if !strings.Contains(pattern, "*") {
		return pattern == value
	}

	parts := strings.Split(pattern, "*")
	position := 0
	if parts[0] != "" {
		if !strings.HasPrefix(value, parts[0]) {
			return false
		}
		position = len(parts[0])
	}

	for _, part := range parts[1 : len(parts)-1] {
		if part == "" {
			continue
		}
		next := strings.Index(value[position:], part)
		if next < 0 {
			return false
		}
		position += next + len(part)
	}

	last := parts[len(parts)-1]
	if last == "" {
		return true
	}
	next := strings.Index(value[position:], last)
	return next >= 0 && strings.HasSuffix(value[position+next:], last)
}
