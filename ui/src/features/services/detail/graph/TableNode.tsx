import { Handle, Position, type Node, type NodeProps } from "@xyflow/react";
import { Loader2, Maximize2, Table2 } from "lucide-react";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";

/**
 * Data carried by a table node. The index signature satisfies React Flow's
 * `Record<string, unknown>` node-data constraint; the named fields are what the
 * graph and this component actually use. Callbacks live here so the node can
 * trigger expansion without the canvas threading props through React Flow.
 */
export interface TableNodeData {
  schema: string;
  table: string;
  /** True once this node's neighbours have been loaded. */
  expanded: boolean;
  /** True while its relations are being fetched. */
  loading: boolean;
  /** Number of relations found, shown once expanded. */
  relationCount?: number;
  /** Expand the node: load its neighbours and open its data panel. */
  onExpand: () => void;
  [key: string]: unknown;
}

export type TableFlowNode = Node<TableNodeData, "table">;

/**
 * A table on the relationship canvas. Shows its qualified name and, once
 * expanded, how many relations it has; the Expand action loads the connected
 * tables as new nodes and opens the data panel for this one. Left/right handles
 * anchor the foreign-key edges drawn between nodes.
 */
export function TableNode({ data, selected }: NodeProps<TableFlowNode>) {
  return (
    <div
      className={cn(
        "w-56 rounded-lg border border-border bg-background shadow-sm transition-shadow",
        selected ? "ring-2 ring-primary" : "hover:shadow-md",
      )}
    >
      <Handle
        type="target"
        position={Position.Left}
        className="!size-2 !border-0 !bg-muted-foreground"
      />
      <div className="flex items-center gap-2 border-b border-border px-3 py-2">
        <Table2 className="size-4 shrink-0 text-muted-foreground" aria-hidden />
        <span className="truncate font-mono text-sm" title={`${data.schema}.${data.table}`}>
          <span className="text-muted-foreground">{data.schema}.</span>
          {data.table}
        </span>
      </div>
      <div className="flex items-center justify-between gap-2 px-3 py-2">
        {data.expanded ? (
          <Badge variant="secondary" className="font-normal">
            {data.relationCount ?? 0}{" "}
            {data.relationCount === 1 ? "relation" : "relations"}
          </Badge>
        ) : (
          <span className="text-xs text-muted-foreground">Not expanded</span>
        )}
        <Button
          type="button"
          size="sm"
          variant="ghost"
          className="h-7 gap-1.5 px-2 text-xs"
          onClick={data.onExpand}
          disabled={data.loading}
        >
          {data.loading ? (
            <Loader2 className="size-3.5 animate-spin" aria-hidden />
          ) : (
            <Maximize2 className="size-3.5" aria-hidden />
          )}
          {data.expanded ? "Data" : "Expand"}
        </Button>
      </div>
      <Handle
        type="source"
        position={Position.Right}
        className="!size-2 !border-0 !bg-muted-foreground"
      />
    </div>
  );
}
