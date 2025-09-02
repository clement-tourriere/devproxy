package caddy

import (
	"encoding/json"
	"fmt"

	"devproxy/internal/docker"
)

type CaddyConfig struct {
	Apps CaddyApps `json:"apps"`
}

type CaddyApps struct {
	HTTP CaddyHTTP `json:"http"`
	TLS  CaddyTLS  `json:"tls"`
}

type CaddyHTTP struct {
	Servers map[string]CaddyServer `json:"servers"`
}

type CaddyTLS struct {
	Automation CaddyTLSAutomation `json:"automation"`
}

type CaddyTLSAutomation struct {
	Policies []CaddyTLSPolicy `json:"policies"`
}

type CaddyTLSPolicy struct {
	Subjects []string                 `json:"subjects"`
	Issuers  []CaddyTLSInternalIssuer `json:"issuers"`
}

type CaddyTLSInternalIssuer struct {
	Module string `json:"module"`
	CA     string `json:"ca,omitempty"`
}

type CaddyServer struct {
	Listen []string     `json:"listen"`
	Routes []CaddyRoute `json:"routes"`
}

type CaddyRoute struct {
	Match    []CaddyMatch   `json:"match"`
	Handle   []CaddyHandler `json:"handle"`
	Terminal bool           `json:"terminal,omitempty"`
}

type CaddyMatch struct {
	Host []string `json:"host"`
}

type CaddyHandler struct {
	Handler   string          `json:"handler"`
	Upstreams []CaddyUpstream `json:"upstreams,omitempty"`
	Headers   *CaddyHeaders   `json:"headers,omitempty"`
}

type CaddyUpstream struct {
	Dial string `json:"dial"`
}

type CaddyHeaders struct {
	Request *CaddyHeadersOps `json:"request,omitempty"`
}

type CaddyHeadersOps struct {
	Set map[string][]string `json:"set,omitempty"`
}

type ConfigGenerator struct{}

func NewConfigGenerator() *ConfigGenerator {
	return &ConfigGenerator{}
}

func (g *ConfigGenerator) GenerateConfig(targets []docker.ProxyTarget) (*CaddyConfig, error) {
	config := &CaddyConfig{
		Apps: CaddyApps{
			HTTP: CaddyHTTP{
				Servers: map[string]CaddyServer{
					"devproxy": {
						Listen: []string{":80", ":443"},
						Routes: g.generateRoutes(targets),
					},
				},
			},
			TLS: CaddyTLS{
				Automation: CaddyTLSAutomation{
					Policies: []CaddyTLSPolicy{
						{
							Subjects: []string{"*.localhost"},
							Issuers: []CaddyTLSInternalIssuer{
								{
									Module: "internal",
									CA:     "local",
								},
							},
						},
					},
				},
			},
		},
	}

	return config, nil
}

func (g *ConfigGenerator) generateRoutes(targets []docker.ProxyTarget) []CaddyRoute {
	var routes []CaddyRoute

	// Group targets by domain
	domainTargets := make(map[string][]docker.ProxyTarget)
	for _, target := range targets {
		domainTargets[target.Domain] = append(domainTargets[target.Domain], target)
	}

	// Create route for each domain
	for domain, targets := range domainTargets {
		// Use the first target (in case of multiple targets for same domain)
		target := targets[0]

		route := CaddyRoute{
			Match: []CaddyMatch{
				{
					Host: []string{domain},
				},
			},
			Handle: []CaddyHandler{
				{
					Handler: "reverse_proxy",
					Upstreams: []CaddyUpstream{
						{
							Dial: fmt.Sprintf("%s:%d", target.ContainerIP, target.Port),
						},
					},
					Headers: &CaddyHeaders{
						Request: &CaddyHeadersOps{
							Set: map[string][]string{
								"Host":              {domain},
								"X-Forwarded-For":   {"{http.request.remote_host}"},
								"X-Forwarded-Proto": {"https"},
								"X-Real-IP":         {"{http.request.remote_host}"},
							},
						},
					},
				},
			},
			Terminal: true,
		}

		routes = append(routes, route)
	}

	return routes
}

func (g *ConfigGenerator) SerializeConfig(config *CaddyConfig) ([]byte, error) {
	return json.MarshalIndent(config, "", "  ")
}
