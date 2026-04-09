import { describe, expect, test } from 'vitest';

import { filterLabelsByPrefixes } from './label-utils';

describe('filterLabelsByPrefixes', () => {
  test('returns empty array when prefixes is empty', () => {
    const labels = { 'universe.engineer/team': 'platform' };
    expect(filterLabelsByPrefixes(labels, [])).toEqual([]);
  });

  test('returns empty array when no labels match', () => {
    const labels = { 'other.io/team': 'platform' };
    expect(filterLabelsByPrefixes(labels, ['universe.engineer/'])).toEqual([]);
  });

  test('strips matching prefix from key', () => {
    const labels = { 'universe.engineer/team': 'platform' };
    expect(filterLabelsByPrefixes(labels, ['universe.engineer/'])).toEqual([
      { key: 'team', value: 'platform' }
    ]);
  });

  test('matches against multiple prefixes', () => {
    const labels = {
      'universe.engineer/team': 'platform',
      'example.com/domain': 'orders'
    };
    expect(
      filterLabelsByPrefixes(labels, ['universe.engineer/', 'example.com/'])
    ).toEqual([
      { key: 'team', value: 'platform' },
      { key: 'domain', value: 'orders' }
    ]);
  });

  test('excludes labels not matching any prefix', () => {
    const labels = {
      'universe.engineer/team': 'platform',
      'kargo.akuity.io/shard': 'default'
    };
    expect(filterLabelsByPrefixes(labels, ['universe.engineer/'])).toEqual([
      { key: 'team', value: 'platform' }
    ]);
  });

  test('handles label where key equals prefix exactly', () => {
    const labels = { 'universe.engineer/': 'value' };
    expect(filterLabelsByPrefixes(labels, ['universe.engineer/'])).toEqual([
      { key: '', value: 'value' }
    ]);
  });

  test('returns empty array when labels is empty', () => {
    expect(filterLabelsByPrefixes({}, ['universe.engineer/'])).toEqual([]);
  });
});
