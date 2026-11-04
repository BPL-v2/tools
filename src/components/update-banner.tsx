import { relaunch } from "@tauri-apps/plugin-process";
import { check, type Update } from "@tauri-apps/plugin-updater";
import { useEffect, useState } from "react";

/** How often a running app looks for a new release. */
const CHECK_INTERVAL_MS = 6 * 60 * 60 * 1000;

/**
 * Checks latest.json of the newest GitHub release and offers a signed
 * in-place update. Renders nothing when up to date or when the check fails
 * (e.g. offline, or a .deb/.rpm install the updater cannot replace).
 */
export function UpdateBanner() {
  const [update, setUpdate] = useState<Update | null>(null);
  const [dismissed, setDismissed] = useState(false);
  const [progress, setProgress] = useState<number | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let cancelled = false;

    const run = async () => {
      try {
        const found = await check();
        if (cancelled) {
          await found?.close();
          return;
        }
        setUpdate((previous) => {
          if (previous && previous !== found) void previous.close();
          return found;
        });
        if (found) setDismissed(false);
      } catch (e) {
        console.warn("Update check failed", e);
      }
    };

    void run();
    const timer = window.setInterval(run, CHECK_INTERVAL_MS);
    return () => {
      cancelled = true;
      window.clearInterval(timer);
    };
  }, []);

  if (!update || dismissed) return null;

  const install = async () => {
    setError(null);
    setProgress(0);
    try {
      let total = 0;
      let done = 0;
      await update.downloadAndInstall((event) => {
        if (event.event === "Started") total = event.data.contentLength ?? 0;
        if (event.event === "Progress") {
          done += event.data.chunkLength;
          if (total) setProgress(Math.round((done / total) * 100));
        }
      });
      // Windows: the installer has already closed the app. Elsewhere: restart into the new version.
      await relaunch();
    } catch (e) {
      setProgress(null);
      setError(String(e));
    }
  };

  const busy = progress !== null;

  return (
    <div role="alert" className="mb-4 alert alert-info">
      <div className="flex-1">
        <div className="font-semibold">
          Version {update.version} is available (you have{" "}
          {update.currentVersion})
        </div>
        {error && <div className="text-sm">Update failed: {error}</div>}
        {busy && (
          <progress
            className="progress w-56"
            value={progress || undefined}
            max={100}
          />
        )}
      </div>
      <div className="flex gap-2">
        <button
          type="button"
          className="btn btn-ghost btn-sm"
          disabled={busy}
          onClick={() => setDismissed(true)}
        >
          Later
        </button>
        <button
          type="button"
          className="btn btn-primary btn-sm"
          disabled={busy}
          onClick={install}
        >
          {busy ? "Installing…" : "Update and restart"}
        </button>
      </div>
    </div>
  );
}
