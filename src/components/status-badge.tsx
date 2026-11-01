import clsx from "clsx";
import type { Status } from "../lib/api";
import { formatTime } from "../lib/format";

function statusLabel(status: Status) {
  switch (status.state) {
    case "running":
      return "Running";
    case "waiting":
      return `Next run at ${formatTime(status.next_run_at)}`;
    case "finished":
      return "Finished";
    case "stopped":
      return "Stopped";
    case "failed":
      return "Failed";
  }
}

export function StatusBadge({ status }: { status: Status | null }) {
  if (!status) return null;
  return (
    <span
      className={clsx("badge", {
        "badge-primary": status.state === "running",
        "badge-info": status.state === "waiting",
        "badge-success": status.state === "finished",
        "badge-neutral": status.state === "stopped",
        "badge-error": status.state === "failed",
      })}
    >
      {statusLabel(status)}
    </span>
  );
}
