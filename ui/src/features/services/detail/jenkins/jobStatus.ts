// Maps a Jenkins job's raw status color to a presentation state — a label and a
// dot colour — the way the LocalStack page's status.ts maps a service's raw
// health state. Jenkins encodes a job's last-build outcome in its `color`, with
// a "…_anime" suffix while a build is currently running.

export interface JobStatus {
  /** Human label, e.g. "Success". */
  label: string;
  /** Tailwind background class for the status dot. */
  dotClass: string;
  /** True when a build is currently running (the "_anime" suffix). */
  building: boolean;
}

/** Base colour (without the "_anime" suffix) → label + dot colour. */
const STATES: Record<string, { label: string; dotClass: string }> = {
  blue: { label: "Success", dotClass: "bg-emerald-500" },
  green: { label: "Success", dotClass: "bg-emerald-500" },
  red: { label: "Failed", dotClass: "bg-destructive" },
  yellow: { label: "Unstable", dotClass: "bg-amber-500" },
  aborted: { label: "Aborted", dotClass: "bg-muted-foreground" },
  disabled: { label: "Disabled", dotClass: "bg-muted-foreground" },
  notbuilt: { label: "Not built", dotClass: "bg-muted-foreground" },
  grey: { label: "Pending", dotClass: "bg-muted-foreground" },
};

const UNKNOWN = { label: "Unknown", dotClass: "bg-muted-foreground" };

/** Resolve a Jenkins job color into a presentation state. */
export function jobStatusFor(color: string): JobStatus {
  const building = color.endsWith("_anime");
  const base = building ? color.slice(0, -"_anime".length) : color;
  const state = STATES[base] ?? UNKNOWN;
  return building
    ? { label: "Building", dotClass: "bg-sky-500", building: true }
    : { ...state, building: false };
}

/** Map a finished build's `result` to a label, for the build history. */
export function buildResultLabel(result: string | undefined, building: boolean): string {
  if (building) return "Building";
  switch (result) {
    case "SUCCESS":
      return "Success";
    case "FAILURE":
      return "Failed";
    case "UNSTABLE":
      return "Unstable";
    case "ABORTED":
      return "Aborted";
    default:
      return result || "—";
  }
}
