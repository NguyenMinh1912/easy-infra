// Public surface of the services feature. The app shell imports the detail
// container and the presentation metadata helper; the add/edit dialog is shared
// with the profiles sidebar, which manages a profile's services. Everything
// else is feature-internal.
export { ServiceDetailPage } from "./detail";
export { metaFor } from "./catalog-meta";
export { ServiceDialog, type DialogState } from "./components/ServiceDialog";
