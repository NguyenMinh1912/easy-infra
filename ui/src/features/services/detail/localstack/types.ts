import type { ServiceInstance } from "@/types/service";

/**
 * Props every AWS service detail page receives. The detail page belongs to a
 * LocalStack service instance; `profile` scopes it when the LocalStack page is
 * profile-scoped (`#/profiles/{p}/services/{s}`).
 */
export interface AwsServiceDetailProps {
  /** The LocalStack service instance hosting this AWS service. */
  service: ServiceInstance;
  /** Set when the LocalStack page is profile-scoped. */
  profile?: string;
}
