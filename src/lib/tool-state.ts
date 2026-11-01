import type {
  InviteReport,
  LogLevel,
  Mismatch,
  RunMode,
  Status,
  StashProgress,
  Tool,
  ToolEventEnvelope,
} from "./api";

const MAX_LOG_LINES = 500;

export interface LogLine {
  id: number;
  level: LogLevel;
  message: string;
  time: number;
}

export interface ToolState {
  status: Status | null;
  mode: RunMode;
  logs: LogLine[];
  nextLogId: number;
  progress: StashProgress | null;
  characterReport: { mismatches: Mismatch[]; time: number } | null;
  inviteReport: { report: InviteReport; time: number } | null;
}

export type ToolStates = Record<Tool, ToolState>;

export type ToolAction =
  | ({ type: "event" } & ToolEventEnvelope)
  | { type: "started"; tool: Tool; mode: RunMode }
  | { type: "start_failed"; tool: Tool; error: string; time: number };

const initialToolState: ToolState = {
  status: null,
  mode: "once",
  logs: [],
  nextLogId: 0,
  progress: null,
  characterReport: null,
  inviteReport: null,
};

export const initialToolStates: ToolStates = {
  characters: initialToolState,
  invites: initialToolState,
  stash: initialToolState,
};

export function isActive(state: ToolState) {
  return state.status?.state === "running" || state.status?.state === "waiting";
}

function appendLog(
  state: ToolState,
  level: LogLevel,
  message: string,
  time: number,
): ToolState {
  const line = { id: state.nextLogId, level, message, time };
  return {
    ...state,
    logs: [...state.logs, line].slice(-MAX_LOG_LINES),
    nextLogId: state.nextLogId + 1,
  };
}

function applyEvent(state: ToolState, { event }: ToolEventEnvelope): ToolState {
  switch (event.type) {
    case "status":
      return { ...state, status: event.status };
    case "log":
      return appendLog(state, event.level, event.message, event.time);
    case "progress": {
      const { type: _, ...progress } = event;
      return { ...state, progress };
    }
    case "character_report":
      return {
        ...state,
        characterReport: { mismatches: event.mismatches, time: event.time },
      };
    case "invite_report":
      return {
        ...state,
        inviteReport: { report: event.report, time: event.time },
      };
  }
}

export function toolReducer(
  states: ToolStates,
  action: ToolAction,
): ToolStates {
  const state = states[action.tool];
  switch (action.type) {
    case "event":
      return { ...states, [action.tool]: applyEvent(state, action) };
    case "started":
      return {
        ...states,
        [action.tool]: { ...state, mode: action.mode, progress: null },
      };
    case "start_failed":
      return {
        ...states,
        [action.tool]: appendLog(state, "error", action.error, action.time),
      };
  }
}
