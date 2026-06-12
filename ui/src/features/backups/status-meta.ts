import {
  Ban,
  CheckCircle2,
  Info,
  Loader2,
  XCircle,
  type LucideIcon,
} from "lucide-react";

import type { BadgeProps } from "@/components/ui/badge";
import type { BackupStatus } from "@/services/api";

/** Presentation for a backup status: label, icon, and badge styling. */
export interface BackupStatusMeta {
  /** Short label shown in the sessions-table badge. */
  label: string;
  /** Longer headline shown in the detail dialog. */
  headline: string;
  icon: LucideIcon;
  /** Badge variant used in the sessions table. */
  variant: NonNullable<BadgeProps["variant"]>;
  /** Whether the icon should spin (running). */
  spin?: boolean;
  /** Extra classes for the icon/header in the detail dialog. */
  className?: string;
}

/**
 * How each backup status renders. Shared by the sessions table (badge) and the
 * detail dialog (header), so the two stay consistent.
 */
export const STATUS_META: Record<BackupStatus, BackupStatusMeta> = {
  running: {
    label: "Running",
    headline: "Backing up…",
    icon: Loader2,
    variant: "secondary",
    spin: true,
  },
  success: {
    label: "Success",
    headline: "Backup complete",
    icon: CheckCircle2,
    variant: "success",
    className: "text-emerald-600 dark:text-emerald-400",
  },
  unsupported: {
    label: "Unsupported",
    headline: "Backup not supported yet",
    icon: Info,
    variant: "outline",
    className: "text-muted-foreground",
  },
  cancelled: {
    label: "Cancelled",
    headline: "Backup cancelled",
    icon: Ban,
    variant: "outline",
    className: "text-muted-foreground",
  },
  error: {
    label: "Failed",
    headline: "Backup failed",
    icon: XCircle,
    variant: "destructive",
    className: "text-destructive",
  },
};
