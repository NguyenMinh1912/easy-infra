import {
  Bell,
  Cloud,
  Code,
  Database,
  FileText,
  Globe,
  HardDrive,
  KeyRound,
  Layers,
  Lock,
  Mail,
  MessagesSquare,
  Network,
  Server,
  Waves,
  Workflow,
} from "lucide-react";
import type { ComponentType } from "react";
import type { LucideIcon } from "lucide-react";

import { SesDetail } from "./SesDetail";
import { SqsDetail } from "./SqsDetail";
import type { AwsServiceDetailProps } from "./types";

/**
 * One AWS service that LocalStack emulates. The `id` matches the canonical AWS
 * name used in the health response and `services` config (e.g. `sqs`), so cards
 * are rendered straight from the live health map. A service ships a `Detail`
 * page only when easy-infra has a resource browser for it; the rest still get a
 * card (icon, name, status) but no chevron to drill into.
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
  Detail?: ComponentType<AwsServiceDetailProps>;
}

/**
 * Catalog of AWS services easy-infra recognises, keyed by canonical id. It
 * supplies presentation only — the live health response decides which cards
 * render. Services with a `Detail` page (SQS, SES today) gain a clickable
 * chevron; add a browser by setting `Detail` here and the overview wires it up
 * without further changes. Unknown services fall back to a generic entry, so a
 * service LocalStack reports that we don't catalog still renders a card.
 */
const CATALOG: AwsServiceApp[] = [
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
  { id: "s3", label: "S3", name: "Simple Storage Service", blurb: "Object storage", icon: HardDrive },
  { id: "sns", label: "SNS", name: "Simple Notification Service", blurb: "Pub/sub messaging", icon: Bell },
  { id: "dynamodb", label: "DynamoDB", name: "DynamoDB", blurb: "NoSQL key-value store", icon: Database },
  { id: "lambda", label: "Lambda", name: "Lambda", blurb: "Serverless functions", icon: Code },
  { id: "kinesis", label: "Kinesis", name: "Kinesis", blurb: "Streaming data", icon: Waves },
  { id: "cloudformation", label: "CloudFormation", name: "CloudFormation", blurb: "Infrastructure as code", icon: Layers },
  { id: "cloudwatch", label: "CloudWatch", name: "CloudWatch", blurb: "Metrics and alarms", icon: FileText },
  { id: "logs", label: "CloudWatch Logs", name: "CloudWatch Logs", blurb: "Log groups and streams", icon: FileText },
  { id: "iam", label: "IAM", name: "Identity and Access Management", blurb: "Users, roles, policies", icon: Lock },
  { id: "sts", label: "STS", name: "Security Token Service", blurb: "Temporary credentials", icon: KeyRound },
  { id: "secretsmanager", label: "Secrets Manager", name: "Secrets Manager", blurb: "Secret storage", icon: KeyRound },
  { id: "ssm", label: "SSM", name: "Systems Manager", blurb: "Parameters and config", icon: Server },
  { id: "kms", label: "KMS", name: "Key Management Service", blurb: "Encryption keys", icon: KeyRound },
  { id: "apigateway", label: "API Gateway", name: "API Gateway", blurb: "HTTP and REST APIs", icon: Globe },
  { id: "ec2", label: "EC2", name: "Elastic Compute Cloud", blurb: "Virtual machines", icon: Server },
  { id: "events", label: "EventBridge", name: "EventBridge", blurb: "Event bus", icon: Network },
  { id: "stepfunctions", label: "Step Functions", name: "Step Functions", blurb: "State machines", icon: Workflow },
  { id: "route53", label: "Route 53", name: "Route 53", blurb: "DNS", icon: Globe },
];

const BY_ID = new Map(CATALOG.map((app) => [app.id, app]));

/** The cataloged AWS service for an id, or undefined if we don't know it. */
export function awsServiceFor(id: string): AwsServiceApp | undefined {
  return BY_ID.get(id);
}

/**
 * Presentation for an AWS service id, falling back to a generic entry (the id
 * upper-cased, a neutral cloud icon, no detail page) so a service LocalStack
 * reports that we don't catalog still renders a card.
 */
export function awsServiceMeta(id: string): AwsServiceApp {
  return (
    BY_ID.get(id) ?? {
      id,
      label: id.toUpperCase(),
      name: id,
      blurb: "AWS service",
      icon: Cloud,
    }
  );
}

/** True when an AWS service has a resource-browser detail page. */
export function hasDetail(id: string): boolean {
  return BY_ID.get(id)?.Detail !== undefined;
}
