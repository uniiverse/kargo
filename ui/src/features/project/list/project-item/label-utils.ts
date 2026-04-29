import { ColorMapHex } from '@ui/features/stage/utils';

const PALETTE = Object.values(ColorMapHex);

/** Returns a deterministic hex color for a label key by hashing the string. */
export function colorForLabelKey(key: string): string {
  let hash = 0;
  for (let i = 0; i < key.length; i++) {
    hash = ((hash << 5) - hash + key.charCodeAt(i)) | 0;
  }
  return PALETTE[Math.abs(hash) % PALETTE.length];
}

/**
 * Given a label map and a list of prefix strings, returns only the labels
 * whose keys begin with one of the given prefixes, with the matching prefix
 * stripped from the key for display.
 *
 * If prefixes is empty, returns an empty array.
 */
export function filterLabelsByPrefixes(
  labels: Record<string, string>,
  prefixes: string[]
): Array<{ key: string; value: string }> {
  if (prefixes.length === 0) {
    return [];
  }
  const result: Array<{ key: string; value: string }> = [];
  for (const [rawKey, value] of Object.entries(labels)) {
    for (const prefix of prefixes) {
      if (rawKey.startsWith(prefix)) {
        result.push({ key: rawKey.slice(prefix.length), value });
        break;
      }
    }
  }
  return result;
}

/** Formats a label as a display string for use in filters and tags. */
export function formatLabel(label: { key: string; value: string }): string {
  return label.value ? `${label.key}: ${label.value}` : label.key;
}

/**
 * Collects all unique display-label strings across multiple label maps.
 * Returns sorted strings suitable for use as filter options.
 */
export function collectUniqueLabels(
  labelMaps: Array<Record<string, string>>,
  prefixes: string[]
): string[] {
  const seen = new Set<string>();
  for (const labels of labelMaps) {
    for (const label of filterLabelsByPrefixes(labels, prefixes)) {
      seen.add(formatLabel(label));
    }
  }
  return Array.from(seen).sort();
}

/**
 * Returns true if a label map contains all of the selected display-label
 * strings. An empty selection matches everything.
 */
export function matchesSelectedLabels(
  labels: Record<string, string>,
  prefixes: string[],
  selectedLabels: string[]
): boolean {
  if (selectedLabels.length === 0) {
    return true;
  }
  const projectLabels = new Set(filterLabelsByPrefixes(labels, prefixes).map(formatLabel));
  return selectedLabels.every((label) => projectLabels.has(label));
}

/**
 * Converts a wire-format label ("rawkey=value") to a display string by
 * stripping the first matching prefix from the key and formatting as
 * "strippedKey: value".
 */
export function wireLabelToDisplay(wire: string, prefixes: string[]): string {
  const eqIdx = wire.indexOf('=');
  if (eqIdx === -1) {
    return wire;
  }
  const rawKey = wire.slice(0, eqIdx);
  const value = wire.slice(eqIdx + 1);
  for (const prefix of prefixes) {
    if (rawKey.startsWith(prefix)) {
      return formatLabel({ key: rawKey.slice(prefix.length), value });
    }
  }
  return formatLabel({ key: rawKey, value });
}
