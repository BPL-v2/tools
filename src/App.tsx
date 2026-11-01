import { Cog6ToothIcon } from "@heroicons/react/24/outline";
import clsx from "clsx";
import { useState } from "react";
import logo from "./assets/bpl-logo.png";
import { CredentialDialog } from "./components/credential-dialog";
import { SettingsPage } from "./components/settings-page";
import { ThemeToggle } from "./components/theme-toggle";
import { ToolPage } from "./components/tool-page";
import type { Tool } from "./lib/api";
import { isActive } from "./lib/tool-state";
import { TOOLS } from "./lib/tools";
import { useTools } from "./lib/use-tools";

type Page = Tool | "settings";

export function App() {
  const [page, setPage] = useState<Page>("characters");
  const { states, start, stop, credentialRequest, clearCredentialRequest } =
    useTools();
  const tool = TOOLS.find(({ id }) => id === page);

  return (
    <div className="flex h-screen bg-base-100 text-base-content">
      <nav className="flex w-64 shrink-0 flex-col border-r-2 border-base-content/20 bg-base-200">
        <div className="flex items-center gap-3 p-4">
          <img src={logo} alt="" className="h-10" />
          <h1 className="text-lg font-bold">BPL Tools</h1>
        </div>
        <ul className="menu w-full flex-1">
          {TOOLS.map(({ id, name }) => (
            <li key={id}>
              <button
                className={clsx(page === id && "menu-active")}
                onClick={() => setPage(id)}
              >
                {name}
                {isActive(states[id]) && (
                  <span
                    className="ml-auto status status-success"
                    aria-label="Running"
                  />
                )}
              </button>
            </li>
          ))}
        </ul>
        <div className="flex items-center justify-between p-2">
          <button
            className={clsx(
              "btn btn-ghost btn-sm",
              page === "settings" && "btn-active",
            )}
            onClick={() => setPage("settings")}
          >
            <Cog6ToothIcon className="size-5" />
            Credentials
          </button>
          <ThemeToggle />
        </div>
      </nav>

      <main className="flex-1 overflow-y-auto p-6">
        {tool ? (
          <ToolPage
            tool={tool}
            state={states[tool.id]}
            onStart={(mode) => void start(tool.id, mode)}
            onStop={() => stop(tool.id)}
          />
        ) : (
          <SettingsPage />
        )}
      </main>

      {credentialRequest && (
        <CredentialDialog
          request={credentialRequest}
          onClose={clearCredentialRequest}
          onSaved={() => {
            const { tool, mode } = credentialRequest.retry;
            clearCredentialRequest();
            void start(tool, mode);
          }}
        />
      )}
    </div>
  );
}
