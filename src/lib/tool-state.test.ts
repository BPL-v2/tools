import { describe, expect, it } from "vitest";
import { initialToolStates, isActive, toolReducer } from "./tool-state";

describe("toolReducer", () => {
  it("routes events to the tool they belong to", () => {
    const next = toolReducer(initialToolStates, {
      type: "event",
      tool: "stash",
      event: { type: "status", status: { state: "running" } },
    });
    expect(isActive(next.stash)).toBe(true);
    expect(isActive(next.invites)).toBe(false);
  });

  it("caps the log and keeps ids unique", () => {
    let states = initialToolStates;
    for (let i = 0; i < 510; i++) {
      states = toolReducer(states, {
        type: "event",
        tool: "invites",
        event: { type: "log", level: "info", message: `${i}`, time: i },
      });
    }
    const { logs } = states.invites;
    expect(logs).toHaveLength(500);
    expect(logs[0].message).toBe("10");
    expect(logs[499].id).toBe(509);
  });

  it("keeps progress without the event tag and clears it on the next start", () => {
    const progress = {
      percent: 50,
      current_time: 1,
      rate_limit: "1/60 per 60s",
      uploaded_entries: 20,
    };
    const running = toolReducer(initialToolStates, {
      type: "event",
      tool: "stash",
      event: { type: "progress", ...progress },
    });
    expect(running.stash.progress).toEqual(progress);

    const restarted = toolReducer(running, {
      type: "started",
      tool: "stash",
      mode: "continuous",
    });
    expect(restarted.stash.progress).toBeNull();
    expect(restarted.stash.mode).toBe("continuous");
  });

  it("logs a failed start", () => {
    const next = toolReducer(initialToolStates, {
      type: "start_failed",
      tool: "characters",
      error: "This tool is already running",
      time: 1,
    });
    expect(next.characters.logs[0].level).toBe("error");
  });
});
