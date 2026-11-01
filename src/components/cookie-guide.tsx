import clsx from "clsx";
import { useState, type ReactNode } from "react";

type Browser = "chromium" | "firefox";

// Drawn rather than screenshotted: a real capture would show a live session
// cookie, and this stays readable in both themes.
const BROWSERS: Record<
  Browser,
  { label: string; tabs: string[]; activeTab: string; steps: string[] }
> = {
  chromium: {
    label: "Chrome / Edge / Brave",
    tabs: ["Elements", "Console", "Sources", "Network"],
    activeTab: "Application",
    steps: [
      "Press F12 (Cmd+Option+I on macOS). Find the Application tab at the top, it may be hidden behind the » menu",
      "In the left sidebar, expand Cookies and click https://www.pathofexile.com",
      "Click the POESESSID row and copy its Value",
    ],
  },
  firefox: {
    label: "Firefox",
    tabs: ["Inspector", "Console", "Debugger", "Network"],
    activeTab: "Storage",
    steps: [
      "Press F12 (Cmd+Option+I on macOS). Open the Storage tab at the top, it may be hidden behind the » menu",
      "In the left sidebar, expand Cookies and click https://www.pathofexile.com",
      "Double-click the Value of the POESESSID row and copy it",
    ],
  },
};

function Marker({ step }: { step: number }) {
  return (
    <span className="badge shrink-0 font-sans badge-sm font-bold badge-primary">
      {step}
    </span>
  );
}

function CookieRow({
  name,
  value,
  highlighted,
}: {
  name: string;
  value: ReactNode;
  highlighted?: boolean;
}) {
  return (
    <div
      className={clsx(
        "grid grid-cols-[7rem_1fr] gap-2 px-2 py-1",
        highlighted && "bg-primary/25 outline-2 outline-primary",
      )}
    >
      <span>{name}</span>
      <span className="flex items-center justify-between gap-2 truncate">
        {value}
      </span>
    </div>
  );
}

export function CookieGuide() {
  const [browser, setBrowser] = useState<Browser>("chromium");
  const { tabs, activeTab, steps } = BROWSERS[browser];

  return (
    <div className="flex flex-col gap-3">
      <div role="tablist" className="tabs tabs-border tabs-sm">
        {(Object.keys(BROWSERS) as Browser[]).map((id) => (
          <button
            key={id}
            type="button"
            role="tab"
            className={clsx("tab", browser === id && "tab-active")}
            onClick={() => setBrowser(id)}
          >
            {BROWSERS[id].label}
          </button>
        ))}
      </div>

      <div
        role="img"
        aria-label={`Illustration of the ${activeTab} tab in the browser dev tools with the POESESSID cookie selected`}
        className="overflow-hidden rounded-box border-2 border-base-content/20 bg-base-100 font-mono text-xs"
      >
        <div className="flex items-center gap-3 border-b border-base-content/20 bg-base-200 px-2 py-1">
          {tabs.map((tab) => (
            <span key={tab} className="opacity-50">
              {tab}
            </span>
          ))}
          <span className="flex items-center gap-1 border-b-2 border-primary font-bold">
            {activeTab}
            <Marker step={1} />
          </span>
        </div>
        <div className="grid grid-cols-[13rem_1fr]">
          <div className="border-r border-base-content/20 p-2">
            <p className="opacity-50">▾ Cookies</p>
            <p className="mt-1 flex items-center gap-1 rounded bg-primary/25 px-1 py-0.5 outline-2 outline-primary">
              <span className="truncate">pathofexile.com</span>
              <Marker step={2} />
            </p>
          </div>
          <div className="min-w-0">
            <div className="grid grid-cols-[7rem_1fr] gap-2 border-b border-base-content/20 px-2 py-1 font-bold">
              <span>Name</span>
              <span>Value</span>
            </div>
            <CookieRow name="cf_clearance" value="•••••••" />
            <CookieRow
              name="POESESSID"
              highlighted
              value={
                <>
                  <span className="truncate">0123456789abcdef…</span>
                  <Marker step={3} />
                </>
              }
            />
          </div>
        </div>
      </div>

      <ol className="flex flex-col gap-1">
        {steps.map((step, index) => (
          <li key={step} className="flex items-start gap-2">
            <Marker step={index + 1} />
            <span>{step}</span>
          </li>
        ))}
      </ol>
    </div>
  );
}
