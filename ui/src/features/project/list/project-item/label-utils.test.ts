import { describe, expect, test } from 'vitest';

import {
  collectUniqueLabels,
  colorForLabelKey,
  filterLabelsByPrefixes,
  formatLabel,
  matchesSelectedLabels
} from './label-utils';

describe('colorForLabelKey', () => {
  test('returns the same color for the same key', () => {
    expect(colorForLabelKey('domain')).toBe(colorForLabelKey('domain'));
  });

  test('returns different colors for different keys', () => {
    expect(colorForLabelKey('domain')).not.toBe(colorForLabelKey('team'));
  });

  test('returns a hex color string', () => {
    expect(colorForLabelKey('anything')).toMatch(/^#[0-9A-Fa-f]{6}$/);
  });
});

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
    expect(filterLabelsByPrefixes(labels, ['universe.engineer/', 'example.com/'])).toEqual([
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

describe('formatLabel', () => {
  test('formats key and value', () => {
    expect(formatLabel({ key: 'team', value: 'platform' })).toBe('team: platform');
  });

  test('returns just key when value is empty', () => {
    expect(formatLabel({ key: 'team', value: '' })).toBe('team');
  });
});

describe('collectUniqueLabels', () => {
  const prefixes = ['universe.engineer/'];

  test('returns empty array for no label maps', () => {
    expect(collectUniqueLabels([], prefixes)).toEqual([]);
  });

  test('deduplicates labels across projects', () => {
    const maps = [
      { 'universe.engineer/team': 'platform' },
      { 'universe.engineer/team': 'platform', 'universe.engineer/domain': 'orders' }
    ];
    expect(collectUniqueLabels(maps, prefixes)).toEqual(['domain: orders', 'team: platform']);
  });

  test('returns sorted results', () => {
    const maps = [{ 'universe.engineer/z': 'last', 'universe.engineer/a': 'first' }];
    expect(collectUniqueLabels(maps, prefixes)).toEqual(['a: first', 'z: last']);
  });
});

describe('matchesSelectedLabels', () => {
  const prefixes = ['universe.engineer/'];

  test('matches everything when selection is empty', () => {
    expect(matchesSelectedLabels({}, prefixes, [])).toBe(true);
  });

  test('matches when project has selected label', () => {
    const labels = { 'universe.engineer/team': 'platform' };
    expect(matchesSelectedLabels(labels, prefixes, ['team: platform'])).toBe(true);
  });

  test('does not match when project lacks selected label', () => {
    const labels = { 'universe.engineer/team': 'platform' };
    expect(matchesSelectedLabels(labels, prefixes, ['domain: orders'])).toBe(false);
  });

  test('requires all selected labels to match', () => {
    const labels = { 'universe.engineer/team': 'platform' };
    expect(matchesSelectedLabels(labels, prefixes, ['team: platform', 'domain: orders'])).toBe(
      false
    );
  });
});
