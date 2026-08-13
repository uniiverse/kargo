/**
 * Static Grafana targets for AnalysisRun verification-log deep-links.
 *
 * Everything points at prod on purpose: there is one Kargo API server, and
 * AnalysisRun jobs always run on the prod platform cluster with their logs in
 * the prod Loki tenant (see PR #3114 and
 * docs/superpowers/specs/2026-08-13-kargo-verification-deeplinks-design.md).
 *
 * If per-environment targeting is ever needed, swap these for a
 * window.__KARGO_GRAFANA_URL__ global injected by the server, the same way
 * config/base-path.ts surfaces window.__KARGO_BASE_PATH__ from an index.html
 * placeholder.
 */

/** Prod Grafana base URL (no trailing slash). */
export const GRAFANA_URL = 'https://monitoring.us-east4.production.universe.engineer';

/** Loki datasource UID, set on the GrafanaDatasource CR (datasource-loki.yaml). */
export const LOKI_DATASOURCE_UID = 'loki';

/** Loki stream label identifying the prod platform cluster where jobs run. */
export const VERIFICATION_CLUSTER = 'platform-us-east4';

/** UID of the provisioned verification-logs dashboard (ZENITH-011 naming). */
export const VERIFICATION_DASHBOARD_UID = 'infra-kargo-verification-logs';
