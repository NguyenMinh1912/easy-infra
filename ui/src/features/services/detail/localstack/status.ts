import type { ServiceState } from "@/types/localstack";

/** Presentation for one service-health state: a dot colour, label, and order. */
interface StatusMeta {
  /** Normalised state. */
  state: ServiceState;
  /** Short label shown in the badge, e.g. "Running". */
  label: string;
  /** Tailwind class for the status dot's fill. */
  dotClass: string;
  /**
   * Sort weight — lower surfaces first, so running/active services lead and
   * disabled ones trail, mirroring the Desktop "what's running" layout.
   */
  order: number;
}

const STATUS: Record<ServiceState, StatusMeta> = {
  running: { state: "running", label: "Running", dotClass: "bg-success", order: 0 },
  available: {
    state: "available",
    label: "Available",
    dotClass: "bg-muted-foreground",
    order: 1,
  },
  error: { state: "error", label: "Error", dotClass: "bg-destructive", order: 2 },
  disabled: {
    state: "disabled",
    label: "Disabled",
    dotClass: "bg-muted-foreground/40",
    order: 3,
  },
};

/**
 * Normalise a raw health state string to a known {@link ServiceState}.
 * LocalStack reports `running`/`available`/`disabled`/`error`; older builds use
 * `starting`/`initialized`, which we fold into the nearest equivalent. Anything
 * unrecognised is treated as `available` so the card still renders.
 */
export function normalizeState(raw: string): ServiceState {
  switch (raw) {
    case "running":
    case "available":
    case "disabled":
    case "error":
      return raw;
    case "starting":
    case "initializing":
      return "available";
    default:
      return "available";
  }
}

/** Presentation metadata for a raw health state string. */
export function statusFor(raw: string): StatusMeta {
  return STATUS[normalizeState(raw)];
}
