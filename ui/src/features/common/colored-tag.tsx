import { Tag } from 'antd';

import { getContrastTextColor } from '@ui/utils/get-contrast-text-color';

import { colorForLabelKey, formatLabel } from '../project/list/project-item/label-utils';

export const ColoredTag = ({ label }: { label: { key: string; value: string } }) => {
  const bgColor = colorForLabelKey(label.key);
  return (
    <Tag className='!mr-0' color={bgColor} style={{ color: getContrastTextColor(bgColor) }}>
      {formatLabel(label)}
    </Tag>
  );
};
