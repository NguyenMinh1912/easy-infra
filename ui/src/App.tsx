import { useEffect, useState } from "react";
import { fetchStatus, type Status } from "./api";

type Load =
  | { state: "loading" }
  | { state: "error"; message: string }
  | { state: "ready"; status: Status };

export default function App() {
  const [load, setLoad] = useState<Load>({ state: "loading" });

  useEffect(() => {
    let active = true;
    fetchStatus()
      .then((status) => active && setLoad({ state: "ready", status }))
      .catch((err) => active && setLoad({ state: "error", message: String(err) }));
    return () => {
      active = false;
    };
  }, []);

  return (
    <main className="app">
      <header className="app__header">
        <h1>easy-infra</h1>
        <p className="app__subtitle">Local/dev infrastructure dashboard</p>
      </header>
      {load.state === "loading" && <p>Loading…</p>}
      {load.state === "error" && (
        <p className="app__error">Could not reach the API: {load.message}</p>
      )}
      {load.state === "ready" && <Dashboard status={load.status} />}
    </main>
  );
}

function Dashboard({ status }: { status: Status }) {
  if (!status.initialized) {
    return (
      <section className="card">
        <h2>No project here</h2>
        <p>
          This folder has no easy-infra project. Run <code>easy-infra init</code>{" "}
          to scaffold one, then refresh.
        </p>
      </section>
    );
  }

  return (
    <div className="grid">
      <section className="card">
        <h2>Active profile</h2>
        <p className="card__lead">{status.activeProfile || "— none —"}</p>
      </section>

      <section className="card">
        <h2>Profiles</h2>
        {status.profiles.length === 0 ? (
          <p>No profiles yet.</p>
        ) : (
          <ul className="list">
            {status.profiles.map((p) => (
              <li key={p.name}>
                {p.name}
                {p.active && <span className="badge">active</span>}
              </li>
            ))}
          </ul>
        )}
      </section>

      <section className="card">
        <h2>Services</h2>
        {status.services.length === 0 ? (
          <p>No services defined.</p>
        ) : (
          <ul className="list">
            {status.services.map((s) => (
              <li key={s}>{s}</li>
            ))}
          </ul>
        )}
      </section>
    </div>
  );
}
