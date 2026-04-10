import { faStar, faUser } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Empty, Flex, Pagination, Select, Space, Tag, Tooltip } from 'antd';
import { useEffect, useMemo, useState } from 'react';

import { LoadingState } from '@ui/features/common';
import { useListProjects } from '@ui/gen/api/v2/core/core';
import { useGetConfig } from '@ui/gen/api/v2/system/system';

import { useLocalStorage } from '../../../utils/use-local-storage';

import { collectUniqueLabels, matchesSelectedLabels } from './project-item/label-utils';
import { ProjectItem } from './project-item/project-item';
import { ProjectListFilter } from './project-list-filter';
import * as styles from './projects-list.module.less';
import { useStarProjects } from './use-star-projects';

const PAGE_SIZE_KEY = 'projects-page-size';
const PAGE_NUMBER_KEY = 'projects-page-number';

export const ProjectsList = () => {
  const [pageSize, setPageSize] = useLocalStorage(PAGE_SIZE_KEY, 10);
  const [page, setPage] = useLocalStorage(PAGE_NUMBER_KEY, 1);
  const [filter, setFilter] = useState('');
  const [starredProjectsView, setStarredProjectsView] = useLocalStorage(
    'starred-projects-view',
    false
  );
  const [myProjectsView, setMyProjectsView] = useLocalStorage('my-projects-view', false);

  const [starred, toggleStar] = useStarProjects();

  const { data: configData } = useGetConfig();
  const projectLabelPrefixes = configData?.data?.projectLabelPrefixes ?? [];
  const [selectedLabels, setSelectedLabels] = useState<string[]>([]);

  const { data, isLoading } = useListProjects({
    pageSize,
    page: page - 1,
    filter,
    uid: starredProjectsView ? starred : undefined,
    mine: myProjectsView || undefined
  });

  const projects = data?.data?.items ?? [];
  const total = data?.data?.total ?? 0;

  useEffect(() => {
    if (total > 0 && page > Math.ceil(total / pageSize)) {
      setPage(Math.ceil(total / pageSize) || 1);
    }
  }, [total, page, pageSize, setPage]);

  const handlePaginationChange = (newPage: number, newPageSize: number) => {
    setPage(newPage);
    setPageSize(newPageSize);
  };

  const handleFilterChange = (newFilter: string) => {
    setFilter(newFilter);
    setPage(1);
  };

  const availableLabels = useMemo(
    () =>
      collectUniqueLabels(
        projects.map((p) => p.metadata?.labels ?? {}),
        projectLabelPrefixes
      ),
    [projects, projectLabelPrefixes]
  );

  const filteredProjects = useMemo(
    () =>
      projects.filter((p) =>
        matchesSelectedLabels(p.metadata?.labels ?? {}, projectLabelPrefixes, selectedLabels)
      ),
    [projects, projectLabelPrefixes, selectedLabels]
  );

  if (isLoading) return <LoadingState />;

  const isEmpty = filteredProjects.length === 0;

  return (
    <>
      <Flex align='center' className={isEmpty ? 'mb-20' : 'mb-6'} gap={8}>
        <ProjectListFilter onChange={handleFilterChange} init={filter} />
        {availableLabels.length > 0 && (
          <Select
            mode='multiple'
            allowClear
            placeholder='Filter by labels'
            options={availableLabels.map((label) => ({ label, value: label }))}
            value={selectedLabels}
            onChange={setSelectedLabels}
            className='min-w-48 max-w-96'
            maxTagCount='responsive'
          />
        )}
        <Space className='ml-auto'>
          <Tooltip title='Shows projects you have been explicitly granted access to. Broad system-level permissions (e.g. kargo-admin) do not qualify.'>
            <Tag.CheckableTag
              checked={myProjectsView}
              onChange={(checked) => {
                setMyProjectsView(checked);
                setPage(1);
              }}
            >
              <FontAwesomeIcon icon={faUser} className='mr-1' />
              My Projects
            </Tag.CheckableTag>
          </Tooltip>
          <Tag.CheckableTag
            checked={starredProjectsView}
            onChange={(checked) => {
              setStarredProjectsView(checked);
              setPage(1);
            }}
          >
            <FontAwesomeIcon icon={faStar} className='mr-1' />
            Starred Projects
          </Tag.CheckableTag>
        </Space>
      </Flex>
      {isEmpty ? (
        <Empty
          description={
            myProjectsView
              ? 'No projects are directly assigned to your account. Disable this filter to see all projects.'
              : undefined
          }
        />
      ) : (
        <>
          <div className={styles.list}>
            {filteredProjects.map((proj) => (
              <ProjectItem
                key={proj?.metadata?.name}
                project={proj}
                starred={starred.includes(proj?.metadata?.uid || '')}
                onToggleStar={(id) => toggleStar(id)}
                projectLabelPrefixes={projectLabelPrefixes}
              />
            ))}
          </div>
          <Flex justify='flex-end' className='mt-8'>
            <Pagination
              total={total}
              className='ml-auto flex-shrink-0'
              pageSize={pageSize}
              current={page}
              onChange={handlePaginationChange}
              showSizeChanger
              hideOnSinglePage
            />
          </Flex>
        </>
      )}
    </>
  );
};
