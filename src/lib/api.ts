import { invoke } from "@tauri-apps/api/core";

export type Tool = "characters" | "invites" | "stash";
export type RunMode = "once" | "continuous";
export type CredentialKind = "poe_session" | "bpl_token";
export type LogLevel = "info" | "warning" | "error";

export type Status =
  | { state: "running" }
  | { state: "waiting"; next_run_at: number }
  | { state: "finished" }
  | { state: "stopped" }
  | { state: "failed"; error: string; credential: CredentialKind | null };

export type Mismatch =
  | { kind: "name"; character: string; level: number; team: string }
  | {
      kind: "ascendancy";
      character: string;
      level: number;
      team: string;
      ascendancy: string;
    };

export interface InviteReport {
  league: string;
  requested: number;
  accepted_members: number;
  newly_accepted: string[];
  unsorted_requests: string[];
  unsorted_members: string[];
}

export interface StashProgress {
  percent: number | null;
  current_time: number;
  rate_limit: string;
  uploaded_entries: number;
}

export type ToolEvent =
  | { type: "status"; status: Status }
  | { type: "log"; level: LogLevel; message: string; time: number }
  | ({ type: "progress" } & StashProgress)
  | { type: "character_report"; mismatches: Mismatch[]; time: number }
  | { type: "invite_report"; report: InviteReport; time: number };

export interface ToolEventEnvelope {
  tool: Tool;
  event: ToolEvent;
}

export interface CredentialStatus {
  poe_session: boolean;
  bpl_token: boolean;
  plaintext_path: string | null;
}

export const credentialStatus = () =>
  invoke<CredentialStatus>("credential_status");

export const saveCredential = (kind: CredentialKind, value: string) =>
  invoke<void>("save_credential", { kind, value });

export const startTool = (tool: Tool, mode: RunMode) =>
  invoke<void>("start_tool", { tool, mode });

export const stopTool = (tool: Tool) => invoke<void>("stop_tool", { tool });
