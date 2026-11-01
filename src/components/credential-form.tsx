import { ArrowTopRightOnSquareIcon } from "@heroicons/react/24/outline";
import { openUrl } from "@tauri-apps/plugin-opener";
import { useState, type FormEvent } from "react";
import { saveCredential, type CredentialKind } from "../lib/api";
import { CREDENTIALS } from "../lib/tools";
import { CookieGuide } from "./cookie-guide";

interface CredentialFormProps {
  kind: CredentialKind;
  submitLabel: string;
  guideOpen?: boolean;
  onSaved: () => void;
  onCancel?: () => void;
}

export function CredentialForm({
  kind,
  submitLabel,
  guideOpen = false,
  onSaved,
  onCancel,
}: CredentialFormProps) {
  const credential = CREDENTIALS[kind];
  const [value, setValue] = useState("");
  const [error, setError] = useState<string | null>(null);

  async function submit(event: FormEvent) {
    event.preventDefault();
    try {
      await saveCredential(kind, value);
      setValue("");
      setError(null);
      onSaved();
    } catch (saveError) {
      setError(String(saveError));
    }
  }

  return (
    <form onSubmit={submit} className="flex flex-col gap-3">
      <details className="collapse-arrow collapse bg-base-300" open={guideOpen}>
        <summary className="collapse-title text-sm font-medium">
          How to get your {credential.name}
        </summary>
        <div className="collapse-content flex flex-col gap-3 text-sm">
          {credential.warning && (
            <p className="text-warning">{credential.warning}</p>
          )}
          <button
            type="button"
            className="btn self-start btn-primary btn-sm"
            onClick={() => void openUrl(credential.link.url)}
          >
            <ArrowTopRightOnSquareIcon className="size-4" />
            {credential.link.label}
          </button>
          {credential.steps && (
            <ol className="list-inside list-decimal space-y-1">
              {credential.steps.map((step) => (
                <li key={step}>{step}</li>
              ))}
            </ol>
          )}
          {kind === "poe_session" && <CookieGuide />}
        </div>
      </details>
      <label className="flex flex-col gap-1">
        <span className="text-sm opacity-70">{credential.description}</span>
        <input
          type="password"
          className="input w-full"
          autoComplete="off"
          value={value}
          onChange={(event) => setValue(event.target.value)}
          placeholder={credential.name}
        />
      </label>
      {error && <p className="text-sm text-error">{error}</p>}
      <div className="flex justify-end gap-2">
        {onCancel && (
          <button type="button" className="btn btn-ghost" onClick={onCancel}>
            Cancel
          </button>
        )}
        <button
          type="submit"
          className="btn btn-primary"
          disabled={!value.trim()}
        >
          {submitLabel}
        </button>
      </div>
    </form>
  );
}
