import { useState, type FormEvent } from "react";
import { Loader2, Plus } from "lucide-react";
import { toast } from "sonner";

import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Input } from "@/components/ui/input";

interface CreateProfileFormProps {
  /** Names of profiles that already exist, for duplicate validation. */
  existingNames: string[];
  onCreate: (name: string) => Promise<void>;
}

/**
 * Form for adding a profile. Owns only its own input/submit state; the actual
 * creation is delegated upward through {@link onCreate}. Validates the name
 * client-side (non-empty, not a duplicate) and surfaces the result as a toast.
 */
export function CreateProfileForm({
  existingNames,
  onCreate,
}: CreateProfileFormProps) {
  const [name, setName] = useState("");
  const [submitting, setSubmitting] = useState(false);

  const trimmed = name.trim();
  const duplicate = existingNames.some(
    (existing) => existing.toLowerCase() === trimmed.toLowerCase(),
  );
  const validationError =
    trimmed.length > 0 && duplicate
      ? `A profile named "${trimmed}" already exists.`
      : null;
  const canSubmit = trimmed.length > 0 && !duplicate && !submitting;

  async function handleSubmit(event: FormEvent) {
    event.preventDefault();
    if (!canSubmit) return;
    setSubmitting(true);
    try {
      await onCreate(trimmed);
      toast.success(`Profile "${trimmed}" created`);
      setName("");
    } catch (cause) {
      toast.error("Could not create profile", {
        description: cause instanceof Error ? cause.message : String(cause),
      });
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle>New profile</CardTitle>
        <CardDescription>
          Creates an empty profile. Add the services you need to it afterwards.
        </CardDescription>
      </CardHeader>
      <CardContent>
        <form onSubmit={handleSubmit} className="flex flex-col gap-3 sm:flex-row">
          <label htmlFor="profile-name" className="sr-only">
            Profile name
          </label>
          <Input
            id="profile-name"
            value={name}
            onChange={(e) => setName(e.target.value)}
            placeholder="e.g. staging-like"
            disabled={submitting}
            aria-invalid={validationError !== null}
            aria-describedby={validationError ? "profile-name-error" : undefined}
            className="sm:max-w-xs"
          />
          <Button type="submit" disabled={!canSubmit}>
            {submitting ? (
              <Loader2 className="animate-spin" aria-hidden />
            ) : (
              <Plus aria-hidden />
            )}
            {submitting ? "Adding…" : "Add profile"}
          </Button>
        </form>
        {validationError && (
          <p
            id="profile-name-error"
            role="alert"
            className="mt-3 text-sm text-destructive"
          >
            {validationError}
          </p>
        )}
      </CardContent>
    </Card>
  );
}
