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
