// Public surface of the services feature. The app shell imports the detail
// container and the presentation metadata helper; the add dialog and the
// settings modal are shared with the profiles sidebar, which manages a
// profile's services. Everything else is feature-internal.
export { ServiceDetailPage, ServiceSettingsDialog } from "./detail";
export { metaFor } from "./catalog-meta";
export {
  ServiceDialog,
  type DialogState,
  type ServiceDialogResult,
} from "./components/ServiceDialog";
