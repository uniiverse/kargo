package config

import (
	"fmt"
	"io/fs"
	"net/http"
	"strings"
	"time"

	"github.com/kelseyhightower/envconfig"
	"k8s.io/client-go/rest"

	"github.com/akuity/kargo/pkg/os"
	"github.com/akuity/kargo/pkg/server/dex"
	"github.com/akuity/kargo/pkg/server/oidc"
	"github.com/akuity/kargo/pkg/types"
)

type StandardConfig struct {
	GracefulShutdownTimeout time.Duration `envconfig:"GRACEFUL_SHUTDOWN_TIMEOUT" default:"30s"`
}

type ServerConfig struct {
	StandardConfig
	// BasePath is the URL path prefix the server is reachable at, normalized
	// to begin with `/` and not end with one (e.g. `/kargo`). When non-empty,
	// every HTTP route the server registers — REST API, ConnectRPC, dex
	// proxy, dashboard SPA — lives under this prefix; the ingress controller
	// in front of the server MUST preserve the prefix (i.e. must NOT strip
	// it). Empty value means the server serves at the root.
	BasePath                    string
	SecretManagementEnabled     bool
	LocalMode                   bool // LocalMode is true if the server is running as a non-containerized process
	TLSConfig                   *TLSConfig
	OIDCConfig                  *oidc.Config
	AdminConfig                 *AdminConfig
	DexProxyConfig              *dex.ProxyConfig
	ArgoCDConfig                ArgoCDConfig
	PermissiveCORSPolicyEnabled bool
	RolloutsIntegrationEnabled  bool
	AnalysisRunLogURLTemplate   string
	AnalysisRunLogToken         string
	AnalysisRunLogHTTPHeaders   map[string]string
	SharedResourcesNamespace    string
	SystemResourcesNamespace    string
	KargoNamespace              string
	// DefaultControllerName is the name of the controller that Stages with no
	// explicit spec.shard are reconciled by. The API server needs to know this
	// only to include it in a get controller heartbeats response so callers will
	// know which controller's liveness to associate with such Stages. The default
	// controller is often unnamed, so an empty string is a valid value.
	DefaultControllerName string
	ProjectLabelPrefixes  []string
	GrafanaConfig         GrafanaConfig
	RestConfig            *rest.Config

	// AdditionalHandlers is a map of path patterns to HTTP handlers that will
	// be registered on the server's HTTP mux alongside its own handlers. This
	// permits downstream consumers to extend the server with additional
	// endpoints.
	AdditionalHandlers map[string]http.Handler

	// DashboardFS, when set, overrides the embedded UI filesystem used by the
	// dashboard handler. The filesystem should contain the built UI assets at
	// its root (i.e., index.html should be at the top level).
	DashboardFS fs.FS
}

// GrafanaConfig holds settings for optional Grafana deep-links surfaced in
// the UI's AnalysisRun logs view (see ui/src/config/grafana.ts). Every field
// defaults to empty; when URL is empty the UI hides the deep-links entirely
// rather than emitting broken links. Values are relayed to the browser via
// window globals injected into index.html at serve time, the same mechanism
// used for ServerConfig.BasePath — see indexHTMLGrafanaURLPlaceholder and
// friends in server.go.
type GrafanaConfig struct {
	// URL is the Grafana base URL, no trailing slash (e.g.
	// https://grafana.example.com).
	URL string `envconfig:"GRAFANA_URL"`
	// LokiDatasourceUID is the UID of the Loki datasource that backs the
	// verification-log dashboard and Explore deep-links.
	LokiDatasourceUID string `envconfig:"GRAFANA_LOKI_DATASOURCE_UID"`
	// VerificationCluster is the value of the Loki `cluster` stream label
	// identifying the cluster where AnalysisRun verification jobs run.
	VerificationCluster string `envconfig:"GRAFANA_VERIFICATION_CLUSTER"`
	// VerificationDashboardUID is the UID of a provisioned Grafana dashboard
	// used for the "Open in Grafana" deep-link.
	VerificationDashboardUID string `envconfig:"GRAFANA_VERIFICATION_DASHBOARD_UID"`
}

func ServerConfigFromEnv() ServerConfig {
	cfg := ServerConfig{}
	envconfig.MustProcess("", &cfg.StandardConfig)
	cfg.SecretManagementEnabled = types.MustParseBool(os.GetEnv("SECRET_MANAGEMENT_ENABLED", "false"))
	if types.MustParseBool(os.GetEnv("TLS_ENABLED", "false")) {
		tlsCfg := TLSConfigFromEnv()
		cfg.TLSConfig = &tlsCfg
	}
	if types.MustParseBool(os.GetEnv("OIDC_ENABLED", "false")) {
		oidcCfg := oidc.ConfigFromEnv()
		cfg.OIDCConfig = &oidcCfg
	}
	if types.MustParseBool(os.GetEnv("ADMIN_ACCOUNT_ENABLED", "false")) {
		adminCfg := AdminConfigFromEnv()
		cfg.AdminConfig = &adminCfg
	}
	if types.MustParseBool(os.GetEnv("DEX_ENABLED", "false")) {
		dexProxyCfg := dex.ProxyConfigFromEnv()
		cfg.DexProxyConfig = &dexProxyCfg
	}
	envconfig.MustProcess("", &cfg.ArgoCDConfig)
	cfg.PermissiveCORSPolicyEnabled =
		types.MustParseBool(os.GetEnv("PERMISSIVE_CORS_POLICY_ENABLED", "false"))
	cfg.RolloutsIntegrationEnabled =
		types.MustParseBool(os.GetEnv("ROLLOUTS_INTEGRATION_ENABLED", "true"))
	if cfg.RolloutsIntegrationEnabled {
		cfg.AnalysisRunLogURLTemplate = os.GetEnv("ANALYSIS_RUN_LOG_URL_TEMPLATE", "")
		cfg.AnalysisRunLogToken = os.GetEnv("ANALYSIS_RUN_LOG_TOKEN", "")
		if headersStr := os.GetEnv("ANALYSIS_RUN_LOG_HTTP_HEADERS", ""); headersStr != "" {
			kvPairs := strings.Split(headersStr, ",")
			cfg.AnalysisRunLogHTTPHeaders = make(map[string]string, len(kvPairs))
			for _, kvPair := range kvPairs {
				kv := strings.SplitN(kvPair, "=", 2)
				if len(kv) != 2 {
					panic(fmt.Sprintf("Invalid key-value pair: %s", kvPair))
				}
				cfg.AnalysisRunLogHTTPHeaders[strings.TrimSpace(kv[0])] = strings.TrimSpace(kv[1])
			}
		}
	}
	cfg.SystemResourcesNamespace = os.GetEnv(
		"SYSTEM_RESOURCES_NAMESPACE",
		"kargo-system-resources",
	)
	cfg.SharedResourcesNamespace = os.GetEnv(
		"SHARED_RESOURCES_NAMESPACE",
		"kargo-shared-resources",
	)
	cfg.KargoNamespace = os.GetEnv("KARGO_NAMESPACE", "kargo")
	cfg.DefaultControllerName = os.GetEnv("DEFAULT_CONTROLLER_NAME", "")
	cfg.BasePath = NormalizeBasePath(os.GetEnv("API_BASE_PATH", ""))
	cfg.ProjectLabelPrefixes = parseProjectLabelPrefixes(
		os.GetEnv("PROJECT_LABEL_PREFIXES", "universe.engineer/"),
	)
	envconfig.MustProcess("", &cfg.GrafanaConfig)
	cfg.GrafanaConfig.URL = normalizeGrafanaURL(cfg.GrafanaConfig.URL)
	return cfg
}

// normalizeGrafanaURL strips a trailing slash from an operator-supplied
// Grafana URL so callers can build deep-links by simple concatenation (e.g.
// url+"/d/"+dashboardUID) without producing a double slash. Empty stays
// empty.
func normalizeGrafanaURL(u string) string {
	return strings.TrimRight(strings.TrimSpace(u), "/")
}

// NormalizeBasePath canonicalizes an operator-supplied basePath: empty stays
// empty; non-empty values are forced to begin with `/` and not end with one.
// Used by both server bootstrap and tests so the normalization is a single
// shared definition.
func NormalizeBasePath(p string) string {
	p = strings.TrimSpace(p)
	if p == "" {
		return ""
	}
	if !strings.HasPrefix(p, "/") {
		p = "/" + p
	}
	return strings.TrimRight(p, "/")
}

func parseProjectLabelPrefixes(raw string) []string {
	if raw == "" {
		return nil
	}
	var prefixes []string
	for _, p := range strings.Split(raw, ",") {
		p = strings.TrimSpace(p)
		if p != "" {
			prefixes = append(prefixes, p)
		}
	}
	return prefixes
}

type TLSConfig struct {
	CertPath string `envconfig:"TLS_CERT_PATH" required:"true"`
	KeyPath  string `envconfig:"TLS_KEY_PATH" required:"true"`
}

func TLSConfigFromEnv() TLSConfig {
	cfg := TLSConfig{}
	envconfig.MustProcess("", &cfg)
	return cfg
}

// AdminConfig represents configuration for an admin account.
type AdminConfig struct {
	// HashedPassword is a bcrypt hash of the password for the admin account.
	HashedPassword string `envconfig:"ADMIN_ACCOUNT_PASSWORD_HASH" required:"true"`
	// TokenIssuer is the value to be used in the ISS claim of ID tokens issued for
	// the admin account.
	TokenIssuer string `envconfig:"ADMIN_ACCOUNT_TOKEN_ISSUER" required:"true"`
	// TokenAudience is the value to be used in the AUD claim of ID tokens issued
	// for the admin account.
	TokenAudience string `envconfig:"ADMIN_ACCOUNT_TOKEN_AUDIENCE" required:"true"`
	// TokenSigningKey is the key used to sign ID tokens for the admin account.
	TokenSigningKey []byte `envconfig:"ADMIN_ACCOUNT_TOKEN_SIGNING_KEY" required:"true"`
	// TokenTTL specifies how long ID tokens for the admin account are valid. i.e.
	// The expiry will be the time of issue plus this duration.
	TokenTTL time.Duration `envconfig:"ADMIN_ACCOUNT_TOKEN_TTL" default:"24h"`
}

// AdminConfigFromEnv returns an AdminConfig populated from environment
// variables.
func AdminConfigFromEnv() AdminConfig {
	var cfg AdminConfig
	envconfig.MustProcess("", &cfg)
	return cfg
}

type ArgoCDURLMap map[string]string

func (a *ArgoCDURLMap) Decode(value string) error {
	urls := make(map[string]string)
	if value != "" {
		pairs := strings.Split(value, ",")
		for _, pair := range pairs {
			pair = strings.TrimSpace(pair)
			if pair == "" {
				continue
			}
			kvpair := strings.SplitN(pair, "=", 2)
			if len(kvpair) != 2 {
				return fmt.Errorf("invalid map item: %q. expected <shard>=<URL>", pair)
			}
			urls[strings.TrimSpace(kvpair[0])] = strings.TrimSpace(kvpair[1])

		}
	}
	*a = ArgoCDURLMap(urls)
	return nil
}

type ArgoCDConfig struct {
	// URLs is a mapping from shard name to Argo CD URL
	URLs ArgoCDURLMap `envconfig:"ARGOCD_URLS"`
}
