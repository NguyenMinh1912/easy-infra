// Client for the easy-infra JSON API served by `easy-infra serve`.

export interface Profile {
  name: string;
  active: boolean;
}

export interface Status {
  initialized: boolean;
  activeProfile: string;
  profiles: Profile[];
  services: string[];
}

export async function fetchStatus(): Promise<Status> {
  const res = await fetch("/api/status");
  if (!res.ok) {
    throw new Error(`status request failed: ${res.status}`);
  }
  return (await res.json()) as Status;
}
