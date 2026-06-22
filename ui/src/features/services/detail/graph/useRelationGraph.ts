import { useCallback, useEffect, useRef, useState } from "react";
import {
  MarkerType,
  useEdgesState,
  useNodesState,
  type Edge,
} from "@xyflow/react";
import { toast } from "sonner";

import { ApiError, getTableRelations } from "@/services/api";
import type { Relation } from "@/types/console";

import type { TableFlowNode } from "./TableNode";

/** Radius (px) at which a newly expanded node's neighbours are placed. */
const RADIUS = 320;

/** A table identified on the canvas by its qualified name. */
export interface TableRef {
  schema: string;
  table: string;
}

/** The node id for a table is its qualified name. */
function nodeId(schema: string, table: string): string {
  return `${schema}.${table}`;
}

/**
 * Owns the relationship canvas's nodes and edges and the logic to grow it:
 * adding a starting table, and expanding a node into the tables it connects to
 * via foreign keys. Expanding also surfaces the node whose data panel should
 * open, so the canvas can show that table's rows alongside the graph.
 */
export function useRelationGraph(profile: string, service: string) {
  const [nodes, setNodes, onNodesChange] = useNodesState<TableFlowNode>([]);
  const [edges, setEdges, onEdgesChange] = useEdgesState<Edge>([]);
  // The table whose data panel is open, or null when it's closed.
  const [openTable, setOpenTable] = useState<TableRef | null>(null);

  // Latest nodes, read by the stable expand callback without depending on them.
  const nodesRef = useRef(nodes);
  nodesRef.current = nodes;

  // Build a node for a table at a position. onExpand is filled in by the caller
  // so it can close over the stable `expand` defined below.
  const makeNode = useCallback(
    (
      schema: string,
      table: string,
      position: { x: number; y: number },
      onExpand: () => void,
    ): TableFlowNode => ({
      id: nodeId(schema, table),
      type: "table",
      position,
      data: { schema, table, expanded: false, loading: false, onExpand },
    }),
    [],
  );

  // The expand action, stable across renders. Loads a node's relations, adds
  // any new neighbours around it, draws the foreign-key edges, and opens the
  // node's data panel.
  const expand = useCallback(
    async (id: string) => {
      const target = nodesRef.current.find((n) => n.id === id);
      if (!target) return;
      const { schema, table } = target.data;
      const origin = target.position;

      // Show the data panel right away; the rows load there independently.
      setOpenTable({ schema, table });

      // Re-expanding an already-loaded node just re-opens its data panel.
      if (target.data.expanded) return;

      setNodes((ns) =>
        ns.map((n) =>
          n.id === id ? { ...n, data: { ...n.data, loading: true } } : n,
        ),
      );

      let relations: Relation[];
      try {
        const res = await getTableRelations(profile, service, schema, table);
        if (res.error) throw new Error(res.error);
        relations = res.relations ?? [];
      } catch (cause) {
        setNodes((ns) =>
          ns.map((n) =>
            n.id === id ? { ...n, data: { ...n.data, loading: false } } : n,
          ),
        );
        toast.error("Couldn't load relations", {
          description: cause instanceof ApiError ? cause.message : String(cause),
        });
        return;
      }

      // The unique neighbour tables not already on the canvas, laid out on an
      // arc around the expanded node.
      const present = new Set(nodesRef.current.map((n) => n.id));
      const newIds = new Set<string>();
      const fresh: { schema: string; table: string }[] = [];
      for (const r of relations) {
        const nid = nodeId(r.schema, r.table);
        if (nid === id || present.has(nid) || newIds.has(nid)) continue;
        newIds.add(nid);
        fresh.push({ schema: r.schema, table: r.table });
      }
      const additions = fresh.map((t, i) => {
        const angle = (Math.PI * 2 * (i + 1)) / (fresh.length + 1) - Math.PI / 2;
        return makeNode(
          t.schema,
          t.table,
          {
            x: origin.x + RADIUS * Math.cos(angle),
            y: origin.y + RADIUS * Math.sin(angle),
          },
          () => expand(nodeId(t.schema, t.table)),
        );
      });

      setNodes((ns) =>
        ns
          .map((n) =>
            n.id === id
              ? {
                  ...n,
                  data: {
                    ...n.data,
                    loading: false,
                    expanded: true,
                    relationCount: relations.length,
                  },
                }
              : n,
          )
          .concat(additions),
      );

      setEdges((es) => {
        const ids = new Set(es.map((e) => e.id));
        const next = [...es];
        for (const r of relations) {
          const other = nodeId(r.schema, r.table);
          // Orient every edge referencing → referenced (FK column → key it
          // points at), regardless of which side was expanded, so the same
          // constraint yields one edge no matter how it's reached.
          const source = r.direction === "references" ? id : other;
          const target = r.direction === "references" ? other : id;
          const label = r.columns
            .map((c) =>
              r.direction === "references"
                ? `${c.local} → ${c.foreign}`
                : `${c.foreign} → ${c.local}`,
            )
            .join(", ");
          const eid = `${source}|${r.constraint}|${target}`;
          if (ids.has(eid)) continue;
          ids.add(eid);
          next.push({
            id: eid,
            source,
            target,
            label,
            labelStyle: { fontSize: 10 },
            markerEnd: { type: MarkerType.ArrowClosed },
          });
        }
        return next;
      });
    },
    [makeNode, profile, service, setEdges, setNodes],
  );

  // Add a starting table to the canvas (or focus it if already present).
  const addTable = useCallback(
    (schema: string, table: string) => {
      const id = nodeId(schema, table);
      if (nodesRef.current.some((n) => n.id === id)) {
        setOpenTable({ schema, table });
        return;
      }
      // Stagger roots so multiple starting points don't stack on each other.
      const roots = nodesRef.current.filter((n) => !n.parentId).length;
      setNodes((ns) =>
        ns.concat(
          makeNode(schema, table, { x: roots * 80, y: roots * 80 }, () =>
            expand(id),
          ),
        ),
      );
    },
    [expand, makeNode, setNodes],
  );

  const reset = useCallback(() => {
    setNodes([]);
    setEdges([]);
    setOpenTable(null);
  }, [setEdges, setNodes]);

  // Drop the data panel's table once its node is removed from the canvas.
  useEffect(() => {
    if (openTable && !nodes.some((n) => n.id === nodeId(openTable.schema, openTable.table))) {
      setOpenTable(null);
    }
  }, [nodes, openTable]);

  return {
    nodes,
    edges,
    onNodesChange,
    onEdgesChange,
    addTable,
    reset,
    openTable,
    setOpenTable,
  };
}
