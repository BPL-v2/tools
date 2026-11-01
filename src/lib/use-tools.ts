import { listen } from "@tauri-apps/api/event";
import { useCallback, useEffect, useReducer, useRef, useState } from "react";
import {
  startTool,
  stopTool,
  type CredentialKind,
  type RunMode,
  type Tool,
  type ToolEventEnvelope,
} from "./api";
import { initialToolStates, toolReducer } from "./tool-state";

export interface CredentialRequest {
  kind: CredentialKind;
  reason: string;
  retry: { tool: Tool; mode: RunMode };
}

export function useTools() {
  const [states, dispatch] = useReducer(toolReducer, initialToolStates);
  const [credentialRequest, setCredentialRequest] =
    useState<CredentialRequest | null>(null);
  // Read from the event listener, which only subscribes once
  const lastModes = useRef<Partial<Record<Tool, RunMode>>>({});

  const start = useCallback(async (tool: Tool, mode: RunMode) => {
    lastModes.current[tool] = mode;
    dispatch({ type: "started", tool, mode });
    try {
      await startTool(tool, mode);
    } catch (error) {
      dispatch({
        type: "start_failed",
        tool,
        error: String(error),
        time: Date.now(),
      });
    }
  }, []);

  const stop = useCallback((tool: Tool) => {
    void stopTool(tool);
  }, []);

  useEffect(() => {
    const unlisten = listen<ToolEventEnvelope>("tool-event", ({ payload }) => {
      dispatch({ type: "event", ...payload });
      const { event, tool } = payload;
      if (
        event.type === "status" &&
        event.status.state === "failed" &&
        event.status.credential
      ) {
        setCredentialRequest({
          kind: event.status.credential,
          reason: event.status.error,
          retry: { tool, mode: lastModes.current[tool] ?? "once" },
        });
      }
    });
    return () => {
      void unlisten.then((stopListening) => stopListening());
    };
  }, []);

  return {
    states,
    start,
    stop,
    credentialRequest,
    clearCredentialRequest: () => setCredentialRequest(null),
  };
}
