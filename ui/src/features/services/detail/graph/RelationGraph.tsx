import { useMemo } from "react";
import {
  Background,
  Controls,
  MiniMap,
  ReactFlow,
  ReactFlowProvider,
} from "@xyflow/react";
import { AlertCircle, Trash2, Workflow } from "lucide-react";

import "@xyflow/react/dist/style.css";

import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import { useAsync } from "@/hooks/useAsync";
import { getSchema } from "@/services/api";

import { AddTableControl } from "./AddTableControl";
import { NodeDataSheet } from "./NodeDataSheet";
import { TableNode } from "./TableNode";
import { useRelationGraph } from "./useRelationGraph";

interface RelationGraphProps {
  /** Profile whose connection the graph explores. */
  profile: string;
  /** Service within the profile (the API path segment). */
  service: string;
}

// Defined once so React Flow doesn't see a new object every render.
const nodeTypes = { table: TableNode };

/**
 * The relationship canvas: pick a table to drop on the board, then expand nodes
 * to grow the graph along foreign keys. Each table is a node; edges run from
 * the referencing column to the key it points at. Expanding a node also opens a
 * side panel with that table's relation query and a preview of its rows.
 */
export function RelationGraph({ profile, service }: RelationGraphProps) {
  const schema = useAsync(
    (signal) => getSchema(profile, service, signal),
    [profile, service],
  );

  const {
    nodes,
    edges,
    onNodesChange,
    onEdgesChange,
    addTable,
    reset,
    openTable,
    setOpenTable,
  } = useRelationGraph(profile, service);

  const tables = useMemo(
    () =>
      schema.status === "success" && !schema.data.error
        ? schema.data.tables
        : [],
    [schema],
  );

  if (schema.status === "loading") {
    return <Skeleton className="h-[70vh] w-full" />;
  }
  const introspectionError =
    schema.status === "error"
      ? schema.error.message
      : schema.status === "success"
        ? schema.data.error
        : undefined;
  if (introspectionError) {
    return (
      <Alert variant="destructive">
        <AlertCircle />
        <div>
          <AlertTitle>Schema unavailable</AlertTitle>
          <AlertDescription className="font-mono text-xs">
            {introspectionError}
          </AlertDescription>
        </div>
      </Alert>
    );
  }

  return (
    <div className="space-y-3">
      <div className="flex items-center justify-between gap-3">
        <AddTableControl tables={tables} onAdd={addTable} />
        <div className="flex items-center gap-3">
          <p className="text-xs text-muted-foreground">
            Add a table, then Expand a node to follow its foreign keys.
          </p>
          {nodes.length > 0 && (
            <Button size="sm" variant="ghost" onClick={reset} className="gap-1.5">
              <Trash2 aria-hidden className="size-3.5" /> Clear
            </Button>
          )}
        </div>
      </div>

      <div className="h-[70vh] overflow-hidden rounded-md border border-border">
        <ReactFlowProvider>
          <ReactFlow
            nodes={nodes}
            edges={edges}
            onNodesChange={onNodesChange}
            onEdgesChange={onEdgesChange}
            nodeTypes={nodeTypes}
            fitView
            proOptions={{ hideAttribution: true }}
          >
            <Background />
            <Controls />
            <MiniMap pannable zoomable />
            {nodes.length === 0 && (
              <div className="pointer-events-none absolute inset-0 flex flex-col items-center justify-center gap-2 text-muted-foreground">
                <Workflow className="size-8" aria-hidden />
                <p className="text-sm">
                  Add a table to start mapping its relationships.
                </p>
              </div>
            )}
          </ReactFlow>
        </ReactFlowProvider>
      </div>

      <NodeDataSheet
        table={openTable}
        profile={profile}
        service={service}
        onOpenChange={(open) => !open && setOpenTable(null)}
      />
    </div>
  );
}
