import { DatabaseBackup, GitFork, RotateCcw, type LucideIcon } from "lucide-react";

import type { BadgeProps } from "@/components/ui/badge";
import type { BackupKind } from "@/services/api";

/** Presentation for a session kind: label, icon, and badge styling. */
export interface BackupKindMeta {
  /** Short label shown in the sessions-table badge. */
  label: string;
  icon: LucideIcon;
  /** Badge variant used in the sessions table. */
  variant: NonNullable<BadgeProps["variant"]>;
}

/**
 * How each session kind renders in the sessions table, so a backup, an apply,
 * and a fork are told apart at a glance.
 */
export const KIND_META: Record<BackupKind, BackupKindMeta> = {
  backup: {
    label: "Backup",
    icon: DatabaseBackup,
    variant: "secondary",
  },
  apply: {
    label: "Restore",
    icon: RotateCcw,
    variant: "outline",
  },
  fork: {
    label: "Fork",
    icon: GitFork,
    variant: "outline",
  },
};
