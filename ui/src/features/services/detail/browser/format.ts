// Display formatting shared by the bucket listing and the object-detail panel.

/** The last path segment of a key or prefix ("a/b/c.txt" -> "c.txt"). */
export function baseName(key: string): string {
  const trimmed = key.endsWith("/") ? key.slice(0, -1) : key;
  const i = trimmed.lastIndexOf("/");
  return i >= 0 ? trimmed.slice(i + 1) : trimmed;
}

/** Human-readable byte size, e.g. "1.2 KB". */
export function formatBytes(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`;
  const units = ["KB", "MB", "GB", "TB"];
  let value = bytes / 1024;
  let unit = 0;
  while (value >= 1024 && unit < units.length - 1) {
    value /= 1024;
    unit += 1;
  }
  return `${value.toFixed(1)} ${units[unit]}`;
}

/** A locale date-time for a modified timestamp, blank when absent/unparseable. */
export function formatTime(value?: string): string {
  if (!value) return "—";
  const date = new Date(value);
  return Number.isNaN(date.getTime()) ? "—" : date.toLocaleString();
}
