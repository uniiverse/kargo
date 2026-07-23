import { Result } from 'antd';
import { generatePath, Navigate, useParams } from 'react-router-dom';

import { paths } from '@ui/config/paths';
import { LoadingState } from '@ui/features/common';
import { useQueryFreightsRest } from '@ui/gen/api/v2/core/core';
import { Freight } from '@ui/gen/api/v2/models';

// Resolves a short, human-supplied identifier to a concrete piece of Freight and
// redirects to its canonical detail page. The identifier is matched, in order of
// precedence, against:
//   1. an exact Freight alias (e.g. "mortal-dragonfly")
//   2. a prefix of the Freight name (its SHA-1 fingerprint)
//   3. a prefix of any Git commit ID the Freight references
//
// (3) is the interesting case: it answers "which Freight was built for this
// commit?", so a short commit SHA (e.g. from a PR comment) links straight to the
// Freight it produced.
const matchFreight = (freights: Freight[], shortSha: string): Freight | undefined => {
  const needle = shortSha.toLowerCase();

  return (
    freights.find((f) => f.alias === shortSha) ??
    freights.find((f) => f.metadata?.name?.toLowerCase().startsWith(needle)) ??
    freights.find((f) => f.commits?.some((c) => c.id?.toLowerCase().startsWith(needle)))
  );
};

export const FreightForRedirect = () => {
  const { name: project = '', shortSha = '' } = useParams<{ name: string; shortSha: string }>();

  const { data, isLoading, error } = useQueryFreightsRest(project);

  if (isLoading) {
    return <LoadingState />;
  }

  // QueryFreights groups results by Warehouse; with no grouping requested the
  // default ('') group holds everything. Flatten across groups to be safe.
  const freights = Object.values(data?.data?.groups ?? {}).flatMap((group) => group.items ?? []);
  const match = matchFreight(freights, shortSha);

  if (error || !match?.metadata?.name) {
    return (
      <Result
        status='404'
        title='Freight not found'
        subTitle={`No Freight in project "${project}" matches "${shortSha}".`}
      />
    );
  }

  return (
    <Navigate
      replace
      to={generatePath(paths.freight, { name: project, freightName: match.metadata.name })}
    />
  );
};

export default FreightForRedirect;
