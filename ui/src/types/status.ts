// Domain models for the easy-infra dashboard. These mirror the JSON contract
// exposed by `easy-infra serve` (GET /api/status) and are the single source of
// truth for the shapes flowing through the UI. The transport layer
// (services/api) maps responses onto these types; UI never re-declares them.

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
