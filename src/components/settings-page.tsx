import { useCallback, useEffect, useState } from "react";
import {
  credentialStatus,
  type CredentialKind,
  type CredentialStatus,
} from "../lib/api";
import { CREDENTIALS } from "../lib/tools";
import { CredentialForm } from "./credential-form";

const KINDS: CredentialKind[] = ["bpl_token", "poe_session"];

export function SettingsPage() {
  const [status, setStatus] = useState<CredentialStatus | null>(null);

  const refresh = useCallback(() => {
    void credentialStatus().then(setStatus);
  }, []);

  useEffect(refresh, [refresh]);

  return (
    <div className="flex max-w-3xl flex-col gap-4">
      <header>
        <h2 className="text-2xl font-bold">Credentials</h2>
        {status && !status.plaintext_path && (
          <p className="opacity-70">
            Stored in your system keychain. Saved values are never shown again.
          </p>
        )}
        {status?.plaintext_path && (
          <div role="alert" className="mt-2 alert alert-soft alert-warning">
            <div>
              <p>
                No system keychain found, so credentials are stored unencrypted
                in this file:
              </p>
              <p className="font-mono text-xs break-all">
                {status.plaintext_path}
              </p>
            </div>
          </div>
        )}
      </header>
      {KINDS.map((kind) => (
        <section key={kind} className="rounded-box bborder bg-base-200 p-4">
          <div className="mb-3 flex items-center gap-2">
            <h3 className="font-semibold">{CREDENTIALS[kind].name}</h3>
            {status && (
              <span
                className={
                  status[kind] ? "badge badge-success" : "badge badge-warning"
                }
              >
                {status[kind] ? "Set" : "Not set"}
              </span>
            )}
          </div>
          <CredentialForm kind={kind} submitLabel="Save" onSaved={refresh} />
        </section>
      ))}
    </div>
  );
}
