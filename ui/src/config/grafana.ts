/**
 * Runtime Grafana targets for AnalysisRun verification-log deep-links.
 *
 * These are injected by the kargo API server at serve time via
 * window.__KARGO_GRAFANA_*__ globals (see ui/index.html and renderIndexHTML
 * in pkg/server/server.go), the same mechanism config/base-path.ts uses for
 * window.__KARGO_BASE_PATH__. The server populates them from GRAFANA_* env
 * vars, so no environment-specific hostname is baked into this bundle.
 *
 * When the server isn't configured with a Grafana URL (the default), every
 * accessor here returns an empty string and grafana-links.ts hides the
 * deep-links rather than emitting broken URLs.
 */

declare global {
  interface Window {
    __KARGO_GRAFANA_URL__?: string;
    __KARGO_LOKI_DATASOURCE_UID__?: string;
    __KARGO_VERIFICATION_CLUSTER__?: string;
    __KARGO_VERIFICATION_DASHBOARD_UID__?: string;
  }
}

/** Grafana base URL (no trailing slash), or '' when deep-links are disabled. */
export const grafanaUrl = (): string => window.__KARGO_GRAFANA_URL__ ?? '';

/** UID of the Loki datasource backing the verification-log dashboard and Explore link. */
export const lokiDatasourceUid = (): string => window.__KARGO_LOKI_DATASOURCE_UID__ ?? '';

/** Loki `cluster` stream label identifying the cluster where verification jobs run. */
export const verificationCluster = (): string => window.__KARGO_VERIFICATION_CLUSTER__ ?? '';

/** UID of the provisioned verification-logs dashboard. */
export const verificationDashboardUid = (): string =>
  window.__KARGO_VERIFICATION_DASHBOARD_UID__ ?? '';

export {};
