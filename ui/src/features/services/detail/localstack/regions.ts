/**
 * Standard AWS regions for the overview's region picker. LocalStack accepts any
 * region string, but offering the canonical set keeps the dropdown predictable
 * and lets us show a friendly geographic name alongside the id.
 */
export interface AwsRegion {
  /** Region id used by the SDK, e.g. `us-east-1`. */
  id: string;
  /** Geographic label, e.g. `US East (N. Virginia)`. */
  label: string;
}

/** The AWS regions offered in the picker, in AWS console order. */
export const AWS_REGIONS: readonly AwsRegion[] = [
  { id: "us-east-1", label: "US East (N. Virginia)" },
  { id: "us-east-2", label: "US East (Ohio)" },
  { id: "us-west-1", label: "US West (N. California)" },
  { id: "us-west-2", label: "US West (Oregon)" },
  { id: "ca-central-1", label: "Canada (Central)" },
  { id: "sa-east-1", label: "South America (São Paulo)" },
  { id: "eu-west-1", label: "Europe (Ireland)" },
  { id: "eu-west-2", label: "Europe (London)" },
  { id: "eu-west-3", label: "Europe (Paris)" },
  { id: "eu-central-1", label: "Europe (Frankfurt)" },
  { id: "eu-north-1", label: "Europe (Stockholm)" },
  { id: "eu-south-1", label: "Europe (Milan)" },
  { id: "ap-south-1", label: "Asia Pacific (Mumbai)" },
  { id: "ap-northeast-1", label: "Asia Pacific (Tokyo)" },
  { id: "ap-northeast-2", label: "Asia Pacific (Seoul)" },
  { id: "ap-northeast-3", label: "Asia Pacific (Osaka)" },
  { id: "ap-southeast-1", label: "Asia Pacific (Singapore)" },
  { id: "ap-southeast-2", label: "Asia Pacific (Sydney)" },
  { id: "ap-east-1", label: "Asia Pacific (Hong Kong)" },
  { id: "me-south-1", label: "Middle East (Bahrain)" },
  { id: "af-south-1", label: "Africa (Cape Town)" },
];

/** The default region when none is persisted or reported by health. */
export const DEFAULT_REGION = "us-east-1";

/** The geographic label for a region id, or the id itself if unknown. */
export function regionLabel(id: string): string {
  return AWS_REGIONS.find((r) => r.id === id)?.label ?? id;
}

/**
 * Human-readable region display, e.g. `US East (N. Virginia) · us-east-1`.
 * Unknown ids show just the id.
 */
export function formatRegion(id: string): string {
  const label = AWS_REGIONS.find((r) => r.id === id)?.label;
  return label ? `${label} · ${id}` : id;
}
