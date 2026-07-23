import { Freight } from '@ui/gen/api/v2/models';

const freightTimestamp = (freight: Freight): number => {
  const raw = freight.metadata?.creationTimestamp ?? freight.discoveredAt;
  return raw ? new Date(raw).getTime() : 0;
};

// Sorts the most recently created Freight first.
const byNewest = (a: Freight, b: Freight): number => freightTimestamp(b) - freightTimestamp(a);

/**
 * Resolves a short, human-supplied identifier to a single piece of Freight.
 *
 * Matching is tiered by precedence:
 *   1. an exact alias (e.g. "mortal-dragonfly")
 *   2. a prefix of the Freight name (its SHA-1 fingerprint)
 *   3. a prefix of any referenced Git commit ID
 *
 * (3) is the motivating case: a short commit SHA resolves to the Freight it
 * produced. Within a tier, when more than one Freight matches, the most
 * recently created one wins.
 */
export const resolveFreightByShortSha = (
  freights: Freight[],
  shortSha: string
): Freight | undefined => {
  if (!shortSha) {
    return undefined;
  }

  const needle = shortSha.toLowerCase();

  const tiers: ((freight: Freight) => boolean)[] = [
    (freight) => freight.alias === shortSha,
    (freight) => Boolean(freight.metadata?.name?.toLowerCase().startsWith(needle)),
    (freight) =>
      Boolean(freight.commits?.some((commit) => commit.id?.toLowerCase().startsWith(needle)))
  ];

  for (const matches of tiers.map((predicate) => freights.filter(predicate).sort(byNewest))) {
    if (matches.length > 0) {
      return matches[0];
    }
  }

  return undefined;
};
