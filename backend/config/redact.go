package config

import (
	"fmt"
	"net"
	"net/url"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

// RedactedConfigMarker starts a redacted config file, so it can't be imported by mistake
const RedactedConfigMarker = "# aura redacted configuration"

// RedactedValue replaces secrets and personal values in a redacted config
const RedactedValue = "<redacted>"

// Keys whose values are secrets or identify the user
var redactedKeyPattern = regexp.MustCompile(`(?i)(token|secret|password|apikey|userkey|hash|webhook|clientid|userid|allowedemails|alloweddomains)`)

var (
	ipv4Pattern  = regexp.MustCompile(`\b(?:\d{1,3}\.){3}\d{1,3}\b`)
	emailPattern = regexp.MustCompile(`[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}`)
)

// RedactConfigYAML replaces the secrets, emails and addresses in a config file, so it can be shared.
// Each address becomes host-1, host-2 and so on, the same address always gets the same name.
// Empty values are kept, so what isn't set still shows.
func RedactConfigYAML(data []byte) ([]byte, error) {
	var doc yaml.Node
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, err
	}
	r := redactor{hosts: map[string]string{}}
	r.node(&doc, false)
	return yaml.Marshal(&doc)
}

type redactor struct {
	hosts map[string]string
}

// node redacts a YAML node, secret is whether a parent key marks everything below it as secret
func (r *redactor) node(n *yaml.Node, secret bool) {
	switch n.Kind {
	case yaml.DocumentNode, yaml.SequenceNode:
		for _, child := range n.Content {
			r.node(child, secret)
		}
	case yaml.MappingNode:
		for i := 0; i+1 < len(n.Content); i += 2 {
			key, value := n.Content[i], n.Content[i+1]
			// Header values often hold credentials
			childSecret := secret || strings.EqualFold(key.Value, "Headers") ||
				(value.Kind != yaml.MappingNode && redactedKeyPattern.MatchString(key.Value))
			r.node(value, childSecret)
		}
	case yaml.ScalarNode:
		if n.Tag != "!!str" || n.Value == "" {
			return
		}
		if secret {
			n.Value = RedactedValue
			return
		}
		n.Value = r.text(n.Value)
	}
}

// text replaces the emails and addresses in a value
func (r *redactor) text(value string) string {
	value = emailPattern.ReplaceAllString(value, "redacted@redacted")
	if u, err := url.Parse(value); err == nil && u.Scheme != "" && u.Host != "" {
		u.User = nil
		// Query values can hold tokens
		if u.RawQuery != "" {
			u.RawQuery = "redacted"
		}
		if port := u.Port(); port != "" {
			u.Host = net.JoinHostPort(r.host(u.Hostname()), port)
		} else {
			u.Host = r.host(u.Hostname())
		}
		return u.String()
	}
	return ipv4Pattern.ReplaceAllStringFunc(value, r.host)
}

// host returns the shared name for an address, loopback addresses are kept
func (r *redactor) host(host string) string {
	lower := strings.ToLower(host)
	if lower == "localhost" || lower == "::1" || strings.HasPrefix(lower, "127.") {
		return host
	}
	if name, ok := r.hosts[lower]; ok {
		return name
	}
	name := fmt.Sprintf("host-%d", len(r.hosts)+1)
	r.hosts[lower] = name
	return name
}
