import { useState, type FormEvent } from "react";
import { Plus } from "lucide-react";

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
  onCreate: (name: string) => Promise<void>;
}

/**
 * Form for adding a profile. Owns only its own input/submit state; the actual
 * creation is delegated upward through {@link onCreate}.
 */
export function CreateProfileForm({ onCreate }: CreateProfileFormProps) {
  const [name, setName] = useState("");
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const trimmed = name.trim();

  async function handleSubmit(event: FormEvent) {
    event.preventDefault();
    if (!trimmed || submitting) return;
    setSubmitting(true);
    setError(null);
    try {
      await onCreate(trimmed);
      setName("");
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : String(cause));
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle>New profile</CardTitle>
        <CardDescription>
          Scaffolds a profile with default config for every service the project
          defines.
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
            className="sm:max-w-xs"
          />
          <Button type="submit" disabled={!trimmed || submitting}>
            <Plus aria-hidden />
            {submitting ? "Adding…" : "Add profile"}
          </Button>
        </form>
        {error && (
          <p role="alert" className="mt-3 text-sm text-destructive">
            {error}
          </p>
        )}
      </CardContent>
    </Card>
  );
}
