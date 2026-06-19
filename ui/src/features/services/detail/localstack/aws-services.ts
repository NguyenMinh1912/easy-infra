import { Mail, MessagesSquare } from "lucide-react";
import type { ComponentType } from "react";
import type { LucideIcon } from "lucide-react";

import { SesDetail } from "./SesDetail";
import { SqsDetail } from "./SqsDetail";
import type { AwsServiceDetailProps } from "./types";

/**
 * One AWS service that LocalStack emulates and easy-infra ships a detail view
 * for. The `id` matches the canonical AWS name used in the localstack
 * `services` config (e.g. `sqs`), so the launcher can flag which apps a profile
 * actually enables.
 */
export interface AwsServiceApp {
  /** Canonical AWS service id, e.g. `sqs`. */
  id: string;
  /** Short display label, e.g. `SQS`. */
  label: string;
  /** Full product name, e.g. `Simple Queue Service`. */
  name: string;
  /** One-line description shown under the label. */
  blurb: string;
  icon: LucideIcon;
  /** The service's own detail page, shown when its card is opened. */
  Detail: ComponentType<AwsServiceDetailProps>;
}

/**
 * AWS service apps easy-infra supports a detail page for, in launcher order.
 * Add a new app by implementing its detail component and registering it here —
 * the LocalStack overview renders the list without any further wiring.
 */
export const AWS_SERVICES: AwsServiceApp[] = [
  {
    id: "sqs",
    label: "SQS",
    name: "Simple Queue Service",
    blurb: "Managed message queues",
    icon: MessagesSquare,
    Detail: SqsDetail,
  },
  {
    id: "ses",
    label: "SES",
    name: "Simple Email Service",
    blurb: "Email sending and receiving",
    icon: Mail,
    Detail: SesDetail,
  },
];

/** The AWS service app with the given id, or undefined if unsupported. */
export function awsServiceFor(id: string): AwsServiceApp | undefined {
  return AWS_SERVICES.find((app) => app.id === id);
}
