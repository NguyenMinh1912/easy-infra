import { Toaster as Sonner, type ToasterProps } from "sonner";

import { useTheme } from "@/components/theme/ThemeProvider";

/**
 * Toast host. Mounted once at the app shell; trigger toasts anywhere via
 * `toast()` from `sonner`. Styling is driven by the design tokens so toasts
 * match the rest of the UI in both light and dark.
 */
export function Toaster(props: ToasterProps) {
  const { resolvedTheme } = useTheme();

  return (
    <Sonner
      theme={resolvedTheme}
      className="toaster group"
      toastOptions={{
        classNames: {
          toast:
            "group toast group-[.toaster]:bg-card group-[.toaster]:text-card-foreground group-[.toaster]:border-border group-[.toaster]:shadow-lg",
          description: "group-[.toast]:text-muted-foreground",
          actionButton:
            "group-[.toast]:bg-primary group-[.toast]:text-primary-foreground",
          cancelButton:
            "group-[.toast]:bg-muted group-[.toast]:text-muted-foreground",
          error:
            "group-[.toaster]:!text-destructive group-[.toaster]:!border-destructive/30",
          success: "group-[.toaster]:!text-success",
        },
      }}
      {...props}
    />
  );
}
