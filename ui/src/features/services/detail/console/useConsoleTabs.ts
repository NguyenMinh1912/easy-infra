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
 * navigation and reloads. Resets to the target connection's tabs when the
 * profile/service changes, so connections never share a tab list.
 */
export function useConsoleTabs(profile: string, service: string) {
  const key = storageKey(profile, service);
  const [state, setState] = useState<ConsoleTabsState>(() =>
    load(profile, service),
  );

  // Switch to the new connection's tabs *during* render, not in an effect.
  // Done in an effect, one render would commit with the previous connection's
  // `state` but the new connection's `key`, and the persist effect below would
  // write the old tabs into the new connection's storage — leaking tabs
  // between profiles whose service ids match (e.g. each profile's first
  // "postgres"). Adjusting state here keeps `state` and `key` in lockstep so
  // the persist always writes the connection it belongs to. See react.dev,
  // "You Might Not Need an Effect" (resetting state when a prop changes).
  const [activeKey, setActiveKey] = useState(key);
  if (activeKey !== key) {
    setActiveKey(key);
    setState(load(profile, service));
  }

  // Persist on every change.
  useEffect(() => {
    try {
      localStorage.setItem(key, JSON.stringify(state));
    } catch {
      // Storage unavailable (private mode, quota) — keep working in memory.
    }
  }, [key, state]);

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

  // Open a console pre-filled with a statement (e.g. double-clicking a table in
  // the sidebar), making it active. Returns the new tab's id so the caller can,
  // for instance, trigger an initial run.
  const openTab = useCallback((title: string, sql: string) => {
    const tab: ConsoleTab = { id: crypto.randomUUID(), title, sql };
    setState((s) => ({ tabs: [...s.tabs, tab], activeId: tab.id }));
    return tab.id;
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
    openTab,
    removeTab,
    updateSql,
  };
}
