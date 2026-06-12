import {
  Activity,
  Eraser,
  Play,
  Save,
  Settings2,
  type LucideIcon,
} from "lucide-react";
import { toast } from "sonner";

import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import type { ServiceDefinition } from "@/types/service";

interface ServiceActionsProps {
  service: ServiceDefinition;
}

/** One operation offered in the service action menu. */
interface ServiceAction {
  id: string;
  label: string;
  icon: LucideIcon;
  /** Styled and confirmed differently — `clean` discards data. */
  destructive?: boolean;
}

const ACTIONS: ServiceAction[] = [
  { id: "status", label: "Service status", icon: Activity },
  { id: "backup", label: "Backup", icon: Save },
  { id: "apply", label: "Apply", icon: Play },
  { id: "clean", label: "Clean", icon: Eraser, destructive: true },
];

/**
 * The navbar action menu for a single service: status, backup, apply, and
 * clean, grouped behind one icon trigger. The operations run server-side
 * (`easy-infra apply`/`backup`/…) which the API does not yet expose, so each
 * item announces that it is coming rather than calling a missing endpoint.
 * Wiring a real action later is a one-line swap of its handler.
 */
export function ServiceActions({ service }: ServiceActionsProps) {
  const run = (action: ServiceAction) => {
    toast.info(`"${action.label}" is not available yet`, {
      description: `Running ${action.id} for ${service.name} from the UI is coming soon.`,
    });
  };

  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button variant="outline" size="icon" aria-label="Service actions">
          <Settings2 />
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end">
        <DropdownMenuLabel>Actions</DropdownMenuLabel>
        <DropdownMenuSeparator />
        {ACTIONS.map((action) => {
          const Icon = action.icon;
          return (
            <DropdownMenuItem
              key={action.id}
              variant={action.destructive ? "destructive" : "default"}
              onSelect={() => run(action)}
            >
              <Icon aria-hidden />
              {action.label}
            </DropdownMenuItem>
          );
        })}
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
