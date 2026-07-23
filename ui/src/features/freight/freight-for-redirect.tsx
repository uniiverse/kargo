import { Result } from 'antd';
import { generatePath, Navigate, useParams } from 'react-router-dom';

import { paths } from '@ui/config/paths';
import { LoadingState } from '@ui/features/common';
import { useQueryFreightsRest } from '@ui/gen/api/v2/core/core';

import { resolveFreightByShortSha } from './resolve-freight-by-short-sha';

// Resolves a short identifier from the URL to a concrete piece of Freight and
// redirects to its canonical detail page. See resolveFreightByShortSha for the
// matching rules; the motivating case is linking a short Git commit SHA (e.g.
// from a release PR comment) straight to the Freight it produced.
export const FreightForRedirect = () => {
  const { name: project = '', shortSha = '' } = useParams<{ name: string; shortSha: string }>();

  const { data, isLoading, error } = useQueryFreightsRest(project);

  if (isLoading) {
    return <LoadingState />;
  }

  // QueryFreights groups results by Warehouse; with no grouping requested the
  // default ('') group holds everything. Flatten across groups to be safe.
  const freights = Object.values(data?.data?.groups ?? {}).flatMap((group) => group.items ?? []);
  const match = resolveFreightByShortSha(freights, shortSha);

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
