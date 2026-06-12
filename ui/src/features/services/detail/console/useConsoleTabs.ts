import { useCallback, useEffect, useState } from "react";

/** A single named SQL console: its title and its (persisted) editor buffer. */
export interface ConsoleTab {
  id: string;
  title: string;
  sql: string;
}

interface ConsoleTabsState {
  tabs: ConsoleTab[];
  activeId: string;
}

/**
 * localStorage key for a profile/service's consoles. Scoped per connection so
 * each postgres service keeps its own set of tabs.
 */
function storageKey(profile: string, service: string): string {
  return `easy-infra:console-tabs:${profile}:${service}`;
}

function newTab(title: string): ConsoleTab {
  return { id: crypto.randomUUID(), title, sql: "" };
}

function initialState(): ConsoleTabsState {
  const tab = newTab("Console 1");
  return { tabs: [tab], activeId: tab.id };
}

/** Read persisted tabs, falling back to a single empty console. */
function load(profile: string, service: string): ConsoleTabsState {
  try {
    const raw = localStorage.getItem(storageKey(profile, service));
    if (!raw) return initialState();
    const parsed = JSON.parse(raw) as Partial<ConsoleTabsState>;
    const tabs = Array.isArray(parsed.tabs)
      ? parsed.tabs.filter(
          (t): t is ConsoleTab =>
            typeof t?.id === "string" &&
            typeof t?.title === "string" &&
            typeof t?.sql === "string",
        )
      : [];
    if (tabs.length === 0) return initialState();
    const activeId = tabs.some((t) => t.id === parsed.activeId)
      ? (parsed.activeId as string)
      : tabs[0].id;
    return { tabs, activeId };
  } catch {
    return initialState();
  }
}

/**
 * Manages a profile/service's SQL consoles — a list of named tabs and the
 * active one — persisted to localStorage so the editor buffers survive
 * navigation and reloads. Re-keys when the connection changes.
 */
export function useConsoleTabs(profile: string, service: string) {
  const [state, setState] = useState<ConsoleTabsState>(() =>
    load(profile, service),
  );

  // Reload when navigating to a different connection's console.
  useEffect(() => {
    setState(load(profile, service));
  }, [profile, service]);

  // Persist on every change.
  useEffect(() => {
    try {
      localStorage.setItem(storageKey(profile, service), JSON.stringify(state));
    } catch {
      // Storage unavailable (private mode, quota) — keep working in memory.
    }
  }, [profile, service, state]);

  const setActive = useCallback((id: string) => {
    setState((s) => (s.tabs.some((t) => t.id === id) ? { ...s, activeId: id } : s));
  }, []);

  const addTab = useCallback(() => {
    setState((s) => {
      // Number from the highest existing "Console N" so titles stay unique
      // even after closing earlier tabs.
      const max = s.tabs.reduce((n, t) => {
        const match = /^Console (\d+)$/.exec(t.title);
        return match ? Math.max(n, Number(match[1])) : n;
      }, 0);
      const tab = newTab(`Console ${max + 1}`);
      return { tabs: [...s.tabs, tab], activeId: tab.id };
    });
  }, []);

  const removeTab = useCallback((id: string) => {
    setState((s) => {
      if (s.tabs.length <= 1) return s; // always keep at least one console
      const index = s.tabs.findIndex((t) => t.id === id);
      const tabs = s.tabs.filter((t) => t.id !== id);
      const activeId =
        s.activeId === id
          ? tabs[Math.min(index, tabs.length - 1)].id
          : s.activeId;
      return { tabs, activeId };
    });
  }, []);

  const updateSql = useCallback((id: string, sql: string) => {
    setState((s) => ({
      ...s,
      tabs: s.tabs.map((t) => (t.id === id ? { ...t, sql } : t)),
    }));
  }, []);

  return {
    tabs: state.tabs,
    activeId: state.activeId,
    setActive,
    addTab,
    removeTab,
    updateSql,
  };
}
