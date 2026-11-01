import { ArrowPathIcon, PlayIcon, StopIcon } from "@heroicons/react/24/solid";
import type { RunMode, Tool } from "../lib/api";
import { isActive, type ToolState } from "../lib/tool-state";
import { LogPanel } from "./log-panel";
import {
  CharacterReportView,
  InviteReportView,
  StashProgressView,
} from "./reports";
import { StatusBadge } from "./status-badge";

interface ToolPageProps {
  tool: { id: Tool; name: string; description: string };
  state: ToolState;
  onStart: (mode: RunMode) => void;
  onStop: () => void;
}

export function ToolPage({ tool, state, onStart, onStop }: ToolPageProps) {
  const active = isActive(state);
  const { status, characterReport, inviteReport, progress } = state;

  return (
    <div className="flex h-full flex-col gap-4">
      <header>
        <div className="flex items-center gap-3">
          <h2 className="text-2xl font-bold">{tool.name}</h2>
          <StatusBadge status={status} />
        </div>
        <p className="opacity-70">{tool.description}</p>
      </header>

      <div className="flex gap-2">
        {active ? (
          <button className="btn btn-error" onClick={onStop}>
            <StopIcon className="size-5" />
            Stop
          </button>
        ) : (
          <>
            <button className="btn btn-primary" onClick={() => onStart("once")}>
              <PlayIcon className="size-5" />
              Run Once
            </button>
            <button
              className="btn btn-secondary"
              onClick={() => onStart("continuous")}
            >
              <ArrowPathIcon className="size-5" />
              Run Continuously
            </button>
          </>
        )}
      </div>

      {status?.state === "failed" && (
        <div role="alert" className="alert alert-soft alert-error">
          {status.error}
        </div>
      )}
      {tool.id === "characters" && characterReport && (
        <CharacterReportView {...characterReport} />
      )}
      {tool.id === "invites" && inviteReport && (
        <InviteReportView {...inviteReport} />
      )}
      {tool.id === "stash" && progress && (
        <StashProgressView progress={progress} />
      )}
      <LogPanel logs={state.logs} />
    </div>
  );
}
