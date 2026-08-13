import { describe, expect, test } from 'vitest';

import { RolloutsAnalysisRun } from '@ui/gen/api/v2/models';

import { buildDashboardUrl, buildExploreUrl } from './grafana-links';

/** A populated, completed AnalysisRun with one job metric. */
function completedRun(): RolloutsAnalysisRun {
  return {
    metadata: { name: 'ar-1', namespace: 'kargo-platform-guinea-pig' },
    status: {
      startedAt: '2026-08-13T10:00:00Z',
      completedAt: '2026-08-13T10:05:00Z',
      metricResults: [
        {
          name: 'availability-check',
          measurements: [
            {
              metadata: {
                'job-name': 'ar-1.availability-check.1',
                'job-namespace': 'kargo-platform-guinea-pig'
              }
            }
          ]
        }
      ]
    }
  };
}

describe('buildDashboardUrl', () => {
  test('builds a per-promotion dashboard URL from a completed run', () => {
    const url = buildDashboardUrl(completedRun(), 'availability-check');
    expect(url).toBe(
      'https://monitoring.us-east4.production.universe.engineer/d/infra-kargo-verification-logs' +
        '?var-job=ar-1.availability-check.1' +
        '&var-namespace=kargo-platform-guinea-pig' +
        '&from=1786615200000' +
        '&to=1786615500000'
    );
  });

  test('returns undefined when run has no status', () => {
    const url = buildDashboardUrl({ metadata: { name: 'ar-1' } }, 'availability-check');
    expect(url).toBeUndefined();
  });

  test('returns undefined when metric is not found', () => {
    const url = buildDashboardUrl(completedRun(), 'non-existent-metric');
    expect(url).toBeUndefined();
  });

  test('returns undefined when run is undefined', () => {
    const url = buildDashboardUrl(undefined, 'availability-check');
    expect(url).toBeUndefined();
  });

  test('uses now as to when run is still in-flight (no completedAt)', () => {
    const run: RolloutsAnalysisRun = {
      ...completedRun(),
      status: {
        ...completedRun().status,
        completedAt: undefined
      }
    };
    const url = buildDashboardUrl(run, 'availability-check');
    expect(url).toContain('&to=now');
  });

  test('returns undefined when job metadata is partial (namespace missing)', () => {
    const run = completedRun();
    // Drop job-namespace, keep job-name.
    delete run.status!.metricResults![0].measurements![0].metadata!['job-namespace'];
    expect(buildDashboardUrl(run, 'availability-check')).toBeUndefined();
  });

  test('returns undefined when startedAt is an unparseable string', () => {
    const run = completedRun();
    run.status!.startedAt = 'not-a-date';
    expect(buildDashboardUrl(run, 'availability-check')).toBeUndefined();
  });
});

describe('buildExploreUrl', () => {
  test('placeholder stub returns undefined', () => {
    expect(buildExploreUrl(completedRun(), 'availability-check')).toBeUndefined();
  });
});
