import { Plus, X } from "lucide-react";

import { Button } from "@/components/ui/button";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";

import { PostgresConsole } from "./PostgresConsole";
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
 */
export function PostgresConsoleTabs({
  profile,
  service,
}: PostgresConsoleTabsProps) {
  const { tabs, activeId, setActive, addTab, removeTab, updateSql } =
    useConsoleTabs(profile, service);

  return (
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
          />
        </TabsContent>
      ))}
    </Tabs>
  );
}
