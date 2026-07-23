import { describe, expect, test } from 'vitest';

import { Freight } from '@ui/gen/api/v2/models';

import { resolveFreightByShortSha } from './resolve-freight-by-short-sha';

const makeFreight = (opts: {
  name?: string;
  alias?: string;
  commitIds?: string[];
  createdAt?: string;
}): Freight =>
  ({
    metadata: { name: opts.name, creationTimestamp: opts.createdAt },
    alias: opts.alias,
    commits: (opts.commitIds ?? []).map((id) => ({ id }))
  }) as unknown as Freight;

describe('resolveFreightByShortSha', () => {
  const freightA = makeFreight({
    name: 'aaaa1111cccc',
    alias: 'mortal-dragonfly',
    commitIds: ['deadbeefcafe1234'],
    createdAt: '2026-07-20T00:00:00Z'
  });
  const freightB = makeFreight({
    name: 'bbbb2222dddd',
    alias: 'nimble-newt',
    commitIds: ['facef00d99998888'],
    createdAt: '2026-07-22T00:00:00Z'
  });

  test('matches an exact alias', () => {
    expect(resolveFreightByShortSha([freightA, freightB], 'nimble-newt')).toBe(freightB);
  });

  test('matches a Freight name prefix', () => {
    expect(resolveFreightByShortSha([freightA, freightB], 'aaaa11')).toBe(freightA);
  });

  test('matches a Git commit ID prefix', () => {
    expect(resolveFreightByShortSha([freightA, freightB], 'facef00d')).toBe(freightB);
  });

  test('commit ID matching is case-insensitive', () => {
    expect(resolveFreightByShortSha([freightA, freightB], 'DEADBEEF')).toBe(freightA);
  });

  test('returns undefined when nothing matches', () => {
    expect(resolveFreightByShortSha([freightA, freightB], 'nomatch')).toBeUndefined();
  });

  test('returns undefined for an empty identifier', () => {
    // An empty needle would otherwise prefix-match everything.
    expect(resolveFreightByShortSha([freightA, freightB], '')).toBeUndefined();
  });

  test('newest wins when several Freight share a commit prefix', () => {
    const older = makeFreight({
      name: 'old',
      commitIds: ['abc123def'],
      createdAt: '2026-07-01T00:00:00Z'
    });
    const newer = makeFreight({
      name: 'new',
      commitIds: ['abc123def'],
      createdAt: '2026-07-10T00:00:00Z'
    });

    // Order in the input array should not affect the result.
    expect(resolveFreightByShortSha([older, newer], 'abc123')).toBe(newer);
    expect(resolveFreightByShortSha([newer, older], 'abc123')).toBe(newer);
  });

  test('alias precedence beats a name/commit prefix on another Freight', () => {
    const aliased = makeFreight({ name: 'zzzz', alias: 'ff00', createdAt: '2026-07-01T00:00:00Z' });
    const prefixed = makeFreight({
      name: 'ff0012345',
      commitIds: ['ff00abcd'],
      createdAt: '2026-07-09T00:00:00Z'
    });

    // Even though `prefixed` is newer, an exact alias match is a higher tier.
    expect(resolveFreightByShortSha([aliased, prefixed], 'ff00')).toBe(aliased);
  });

  test('name-prefix precedence beats a commit prefix on another Freight', () => {
    const namePrefixed = makeFreight({ name: 'cafe1234', createdAt: '2026-07-01T00:00:00Z' });
    const commitPrefixed = makeFreight({
      name: 'zzzz',
      commitIds: ['cafe9999'],
      createdAt: '2026-07-09T00:00:00Z'
    });

    expect(resolveFreightByShortSha([namePrefixed, commitPrefixed], 'cafe')).toBe(namePrefixed);
  });
});
