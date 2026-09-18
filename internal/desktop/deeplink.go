package desktop

import (
	"errors"
	"net/url"
	"strings"
	"unicode"
)

const (
	maxDeepLinkLength = 2048
	maxDeepLinkIDSize = 256
)

// DeepLinkAction identifies the destination and resource targeted by a nexus:// URL.
type DeepLinkAction struct {
	Resource string            `json:"resource"` // "project", "mission", "agent"
	ID       string            `json:"id"`
	Params   map[string]string `json:"params,omitempty"`
}

// ParseDeepLink parses and validates a nexus:// URI.
// Supported schemes:
//
//	nexus://project/<id>
//	nexus://mission/<id>
//	nexus://agent/<id>
func ParseDeepLink(rawURL string) (*DeepLinkAction, error) {
	if len(rawURL) == 0 || len(rawURL) > maxDeepLinkLength {
		return nil, errors.New("deep link length is invalid")
	}

	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, err
	}

	if !strings.EqualFold(u.Scheme, "nexus") || u.User != nil || u.Fragment != "" {
		return nil, errors.New("invalid scheme: expected nexus://")
	}

	resource := strings.ToLower(u.Host)
	segments := splitDeepLinkPath(u.Path)
	if resource == "" {
		if len(segments) < 1 {
			return nil, errors.New("missing deep link resource")
		}
		resource = strings.ToLower(segments[0])
		segments = segments[1:]
	}

	switch resource {
	case "project", "mission", "agent":
		if len(segments) != 1 || !validDeepLinkID(segments[0]) {
			return nil, errors.New("missing resource id in deep link")
		}
	default:
		return nil, errors.New("unsupported deep link resource: " + resource)
	}

	params := make(map[string]string)
	for k, v := range u.Query() {
		if len(v) > 0 {
			params[k] = v[0]
		}
	}

	return &DeepLinkAction{
		Resource: resource,
		ID:       segments[0],
		Params:   params,
	}, nil
}

func splitDeepLinkPath(rawPath string) []string {
	trimmed := strings.Trim(rawPath, "/")
	if trimmed == "" {
		return nil
	}
	parts := strings.Split(trimmed, "/")
	for i := range parts {
		parts[i], _ = url.PathUnescape(parts[i])
	}
	return parts
}

func validDeepLinkID(id string) bool {
	if id == "" || len(id) > maxDeepLinkIDSize || id == "." || id == ".." {
		return false
	}
	for _, r := range id {
		if unicode.IsSpace(r) || unicode.IsControl(r) || r == '/' || r == '\\' {
			return false
		}
	}
	return true
}
