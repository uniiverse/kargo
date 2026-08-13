import {
  GRAFANA_URL,
  LOKI_DATASOURCE_UID,
  VERIFICATION_CLUSTER,
  VERIFICATION_DASHBOARD_UID
} from '@ui/config/grafana';
import { RolloutsAnalysisRun } from '@ui/gen/api/v2/models';

/** Job identity + time window extracted from an AnalysisRun's status. */
type JobContext = {
  jobName: string;
  jobNamespace: string;
  /** epoch ms */
  fromMs: number;
  /** epoch ms, or undefined for an in-flight run (caller uses 'now'). */
  toMs?: number;
};

/**
 * Pulls the promotion-unique job identity and time window for a given metric
 * out of an AnalysisRun's status. Returns undefined when the metric result or
 * job metadata isn't populated yet (in-flight run, or a non-job metric), so
 * callers hide the link rather than emit a broken URL.
 */
function jobContext(
  run: RolloutsAnalysisRun | undefined,
  metricName: string
): JobContext | undefined {
  const status = run?.status;
  if (!status?.startedAt) {
    return undefined;
  }

  const result = status.metricResults?.find((r) => r.name === metricName);
  const metadata = result?.measurements?.[0]?.metadata;
  const jobName = metadata?.['job-name'];
  const jobNamespace = metadata?.['job-namespace'];
  if (!jobName || !jobNamespace) {
    return undefined;
  }

  const fromMs = Date.parse(status.startedAt);
  if (Number.isNaN(fromMs)) {
    return undefined;
  }

  const completedAt = status.completedAt;
  const parsedTo = completedAt ? Date.parse(completedAt) : NaN;
  const toMs = Number.isNaN(parsedTo) ? undefined : parsedTo;

  return { jobName, jobNamespace, fromMs, toMs };
}

/** Grafana time value: an epoch-ms string, or 'now' for an open-ended (in-flight) run. */
function grafanaTime(ms: number | undefined): string {
  return ms === undefined ? 'now' : String(ms);
}

/**
 * A per-promotion link to the curated Grafana dashboard. The dashboard's `job`
 * and `namespace` template vars are set from the AnalysisRun's job identity;
 * the time range is bounded to the run's lifetime.
 */
export const buildDashboardUrl = (
  run: RolloutsAnalysisRun | undefined,
  metricName: string
): string | undefined => {
  const ctx = jobContext(run, metricName);
  if (!ctx) {
    return undefined;
  }

  const params = new URLSearchParams({
    'var-job': ctx.jobName,
    'var-namespace': ctx.jobNamespace,
    from: String(ctx.fromMs),
    to: grafanaTime(ctx.toMs)
  });

  return `${GRAFANA_URL}/d/${VERIFICATION_DASHBOARD_UID}?${params.toString()}`;
};

/**
 * A per-promotion link to Grafana Explore for ad-hoc log work (widen the
 * window, add line filters, pivot). Encodes Grafana's `left` querystring: the
 * Loki datasource, a LogQL query pinned to the promotion pod, and the run's
 * time range.
 *
 * The `?left=<json>` schema is a Grafana-versioned contract. It is validated
 * against Grafana v12 (our prod version); a major Grafana upgrade can change
 * the Explore state encoding and silently break this link. The dashboard link
 * is the primary affordance precisely because it hides this behind a stable
 * /d/<uid> URL. If prod Grafana moves past v12, re-verify this encoding.
 */
export const buildExploreUrl = (
  run: RolloutsAnalysisRun | undefined,
  metricName: string
): string | undefined => {
  const ctx = jobContext(run, metricName);
  if (!ctx) {
    return undefined;
  }

  // Unescaped '.' in the pod regex is intentional: matches the deployed inline
  // query in chart-values.yaml (#3114) so the deep-link and Kargo's inline log
  // view use identical LogQL. Job names are DNS-label-safe, so false matches
  // can't occur in practice.
  const expr =
    `{cluster="${VERIFICATION_CLUSTER}", namespace="${ctx.jobNamespace}"} ` +
    `| pod=~"${ctx.jobName}.*"`;

  const left = {
    datasource: LOKI_DATASOURCE_UID,
    queries: [
      {
        refId: 'A',
        datasource: { type: 'loki', uid: LOKI_DATASOURCE_UID },
        expr
      }
    ],
    range: {
      from: String(ctx.fromMs),
      to: grafanaTime(ctx.toMs)
    }
  };

  return `${GRAFANA_URL}/explore?left=${encodeURIComponent(JSON.stringify(left))}`;
};
