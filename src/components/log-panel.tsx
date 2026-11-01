import clsx from "clsx";
import { useEffect, useRef } from "react";
import { formatTime } from "../lib/format";
import type { LogLine } from "../lib/tool-state";

export function LogPanel({ logs }: { logs: LogLine[] }) {
  const container = useRef<HTMLDivElement>(null);

  useEffect(() => {
    container.current?.scrollTo({ top: container.current.scrollHeight });
  }, [logs]);

  return (
    <div
      ref={container}
      className="min-h-40 flex-1 overflow-y-auto rounded-box bborder bg-base-200 p-3 font-mono text-xs"
    >
      {logs.length === 0 && <p className="opacity-50">No output yet.</p>}
      {logs.map((line) => (
        <p
          key={line.id}
          className={clsx(
            "break-words",
            line.level === "warning" && "text-warning",
            line.level === "error" && "text-error",
          )}
        >
          <span className="opacity-50">{formatTime(line.time)}</span>{" "}
          {line.message}
        </p>
      ))}
    </div>
  );
}
