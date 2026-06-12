import { Plus, X } from "lucide-react";
import { useEffect, useMemo, useState } from "react";

import { Button } from "@/components/ui/button";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { useAsync } from "@/hooks/useAsync";
import { getSchema } from "@/services/api";

import { PostgresConsole } from "./PostgresConsole";
import { SchemaSidebar } from "./SchemaSidebar";
import { useConsoleTabs } from "./useConsoleTabs";

interface PostgresConsoleTabsProps {
  /** Profile whose saved connection config the statements run against. */
  profile: string;
  /** Service name within the profile (the API path segment). */
  service: string;
}

/**
 * Hosts one or more SQL consoles against a profile's postgres as tabs. Each tab
 * is an independent {@link PostgresConsole}; its editor buffer is persisted per
 * connection (see {@link useConsoleTabs}), so consoles survive navigation and
 * reloads. Users add consoles with the "+" action and close any but the last.
 *
 * The schema sidebar belongs to the connection, not to any one console, so it
 * lives here — pinned on the left, shared by every tab — while the tab bar sits
 * in the right column above each console's query results.
 */
export function PostgresConsoleTabs({
  profile,
  service,
}: PostgresConsoleTabsProps) {
  const { tabs, activeId, setActive, addTab, removeTab, updateSql } =
    useConsoleTabs(profile, service);

  // The schema browsed in the sidebar. Defaults to the connection's configured
  // schema once introspection lands (see the effect below).
  const [selectedSchema, setSelectedSchema] = useState<string | null>(null);

  // Fetched once per connection and shared by every console tab; completion
  // degrades to keywords-only while it loads or when introspection fails.
  const schemaState = useAsync(
    (signal) => getSchema(profile, service, signal),
    [profile, service],
  );
  const completionSchema = useMemo(() => {
    if (schemaState.status !== "success" || schemaState.data.error) {
      return undefined;
    }
    // Tables in the connection's current schema (its search_path) complete
    // unqualified, matching how unqualified names resolve when the statement
    // runs; tables in other schemas keep their schema prefix.
    const current = schemaState.data.currentSchema || "public";
    const schema: Record<string, string[]> = {};
    for (const table of schemaState.data.tables) {
      const key =
        table.schema === current
          ? table.name
          : `${table.schema}.${table.name}`;
      schema[key] = table.columns;
    }
    return schema;
  }, [schemaState]);

  // Schema introspection lands either with usable data or an `error` envelope
  // (database unreachable); the sidebar mirrors the latter as "unavailable".
  const schemaInfo =
    schemaState.status === "success" && !schemaState.data.error
      ? schemaState.data
      : null;

  // Distinct schemas the connection can see, plus the configured one even if it
  // holds no tables, so the default selection always appears in the dropdown.
  const schemas = useMemo(() => {
    if (!schemaInfo) return [];
    const names = new Set<string>();
    for (const table of schemaInfo.tables) names.add(table.schema);
    if (schemaInfo.currentSchema) names.add(schemaInfo.currentSchema);
    return Array.from(names).sort();
  }, [schemaInfo]);

  // Default the sidebar to the connection's configured schema; reselect when it
  // changes (a different profile/service was navigated to).
  const currentSchema = schemaInfo?.currentSchema || null;
  useEffect(() => {
    if (currentSchema) setSelectedSchema(currentSchema);
  }, [currentSchema]);

  const tablesInSchema = useMemo(() => {
    if (!schemaInfo || !selectedSchema) return [];
    return schemaInfo.tables.filter((t) => t.schema === selectedSchema);
  }, [schemaInfo, selectedSchema]);

  return (
    <div className="flex gap-4">
      <SchemaSidebar
        loading={schemaState.status === "loading"}
        unavailable={schemaState.status === "success" && !schemaInfo}
        schemas={schemas}
        selected={selectedSchema}
        onSelect={setSelectedSchema}
        tables={tablesInSchema}
      />
      <div className="min-w-0 flex-1">
        <Tabs value={activeId} onValueChange={setActive}>
          <div className="flex items-center gap-2">
            <TabsList variant="line" className="flex-wrap">
              {tabs.map((tab) => (
                <TabsTrigger key={tab.id} value={tab.id} className="pr-1">
                  {tab.title}
                  {tabs.length > 1 && (
                    <span
                      role="button"
                      tabIndex={0}
                      aria-label={`Close ${tab.title}`}
                      className="ml-1 inline-flex size-4 items-center justify-center rounded-sm text-muted-foreground hover:bg-muted hover:text-foreground"
                      onPointerDown={(e) => {
                        // Stop the trigger from also activating the tab we're closing.
                        e.stopPropagation();
                        e.preventDefault();
                        removeTab(tab.id);
                      }}
                      onKeyDown={(e) => {
                        if (e.key === "Enter" || e.key === " ") {
                          e.preventDefault();
                          removeTab(tab.id);
                        }
                      }}
                    >
                      <X className="size-3" aria-hidden />
                    </span>
                  )}
                </TabsTrigger>
              ))}
            </TabsList>
            <Button
              variant="ghost"
              size="icon"
              className="size-7 shrink-0"
              aria-label="Add console"
              onClick={addTab}
            >
              <Plus aria-hidden />
            </Button>
          </div>

          {tabs.map((tab) => (
            <TabsContent key={tab.id} value={tab.id} className="mt-2">
              <PostgresConsole
                profile={profile}
                service={service}
                sql={tab.sql}
                onSqlChange={(sql) => updateSql(tab.id, sql)}
                completionSchema={completionSchema}
                schemaResolved={schemaState.status === "success"}
              />
            </TabsContent>
          ))}
        </Tabs>
      </div>
    </div>
  );
}
