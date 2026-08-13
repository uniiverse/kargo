import {
  GRAFANA_URL,
  // eslint-disable-next-line @typescript-eslint/no-unused-vars
  LOKI_DATASOURCE_UID,
  // eslint-disable-next-line @typescript-eslint/no-unused-vars
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
    to: ctx.toMs === undefined ? 'now' : String(ctx.toMs)
  });

  return `${GRAFANA_URL}/d/${VERIFICATION_DASHBOARD_UID}?${params.toString()}`;
};

/**
 * Placeholder — implemented in Task 3.
 * LOKI_DATASOURCE_UID and VERIFICATION_CLUSTER are used by Task 3's implementation.
 */
/* eslint-disable @typescript-eslint/no-unused-vars */
export const buildExploreUrl = (
  _run: RolloutsAnalysisRun | undefined,
  _metricName: string
): string | undefined => undefined;
/* eslint-enable @typescript-eslint/no-unused-vars */
