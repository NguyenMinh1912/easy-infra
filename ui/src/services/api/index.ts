// Public surface of the API transport layer. Import from "@/services/api".
export { ApiError } from "./client";
export { getStatus } from "./status";
export {
  listServices,
  getServiceCatalog,
  createService,
  updateService,
  deleteService,
  startServiceBackup,
  getBackup,
  cancelBackup,
  listBackups,
  deleteBackup,
  listSnapshots,
  startServiceApply,
  startServiceFork,
} from "./services";
export type {
  BackupStatus,
  BackupSession,
  BackupLog,
  BackupPoll,
  BackupList,
  SnapshotsResponse,
} from "./services";
export { executeQuery, getSchema } from "./console";
export {
  activateProfile,
  checkServiceConnection,
  createProfile,
  deleteProfile,
  getProfileConfig,
  listProfiles,
  updateProfileConfig,
} from "./profiles";
export type { ConnectionCheck } from "./profiles";
