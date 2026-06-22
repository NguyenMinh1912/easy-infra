// Domain types for the Jenkins detail page, mirroring the
// /api/profiles/{name}/services/{service}/info, /jobs and /builds JSON
// contracts. A failing request (server unreachable) is an expected outcome: the
// API responds 200 with `error` set and the payload empty.

/** A Jenkins server's identity and summary state. */
export interface JenkinsInfo {
  /** Running Jenkins version, from the X-Jenkins header, when reported. */
  version?: string;
  /** Controller node name; empty for the built-in node. */
  nodeName?: string;
  /** Controller node description, when set. */
  description?: string;
  /** Controller usage mode ("NORMAL" or "EXCLUSIVE"). */
  mode?: string;
  /** True when the server is preparing to shut down. */
  quietingDown: boolean;
  /** Number of top-level jobs on the server. */
  jobCount: number;
}

/** Response of GET …/info — the instance summary. */
export interface JenkinsInfoResponse extends Partial<JenkinsInfo> {
  error?: string;
}

/** One Jenkins job with its last-build status. */
export interface JobInfo {
  name: string;
  url: string;
  /**
   * Jenkins's raw status color: "blue", "red", "yellow", "disabled",
   * "notbuilt", or a "…_anime" variant while a build runs. The UI maps it to a
   * label and badge (see jobStatus.ts).
   */
  color: string;
  /** Most recent build number, 0/absent when the job has never built. */
  lastBuild?: number;
}

/** Response of GET …/jobs — the profile's Jenkins jobs. */
export interface JobsResponse {
  jobs: JobInfo[];
  error?: string;
}

/** One build of a Jenkins job. */
export interface BuildInfo {
  number: number;
  /** Outcome ("SUCCESS", "FAILURE", "UNSTABLE", "ABORTED"); empty if running. */
  result?: string;
  building: boolean;
  /** Start time in Unix milliseconds. */
  timestamp: number;
  /** Duration in milliseconds, 0 while still running. */
  duration: number;
}

/** Response of GET …/builds — a job's recent builds, most recent first. */
export interface BuildsResponse {
  builds: BuildInfo[];
  error?: string;
}

/**
 * Response of GET …/log — a chunk of a build's progressive console output. The
 * dialog long-polls: it appends `text`, then re-requests from `offset` until
 * `more` is false (the build has finished producing output).
 */
export interface BuildLogResponse {
  /** Console output appended since the requested start offset. */
  text: string;
  /** Byte offset to pass as `start` on the next poll. */
  offset: number;
  /** True while the build is still producing output. */
  more: boolean;
  error?: string;
}
