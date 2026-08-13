import { faExternalLink } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Tag } from 'antd';
import classNames from 'classnames';
import { generatePath, Link } from 'react-router-dom';

import { paths } from '@ui/config/paths';
import { StageConditionType } from '@ui/features/common/stage-status/utils';
import { Stage } from '@ui/gen/api/v2/models';

type AnalysisRunLogsLinkProps = {
  stage: Stage;
  className?: string;
};

export const AnalysisRunLogsLink = (props: AnalysisRunLogsLinkProps) => {
  if (
    props.stage?.status?.conditions?.find(
      (condition) => condition?.type === StageConditionType.Promoting
    )
  ) {
    return null;
  }

  const recentVerification = props.stage?.status?.freightHistory?.[0]?.verificationHistory?.[0];

  // Show the link for any completed verification that produced an AnalysisRun —
  // on-call reads green-run logs too, not just failures. The "not currently
  // Promoting" guard above already hides it mid-promotion.
  const analysisRunName = recentVerification?.analysisRun?.name;
  const completed =
    recentVerification?.phase === 'Successful' || recentVerification?.phase === 'Failed';

  if (!completed || !analysisRunName) {
    return null;
  }

  const logsLink = generatePath(paths.analysisRunLogs, {
    name: props.stage?.metadata?.namespace,
    stageName: props.stage?.metadata?.name,
    analysisRunId: analysisRunName
  });

  return (
    <Link to={logsLink} target='_blank' className={classNames(props.className)}>
      <Tag color='orange' className='text-[10px]' bordered={false}>
        Analysis Run Logs <FontAwesomeIcon icon={faExternalLink} className='ml-1' />
      </Tag>
    </Link>
  );
};
