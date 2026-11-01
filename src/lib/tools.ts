import type { CredentialKind, Tool } from "./api";

export const TOOLS: { id: Tool; name: string; description: string }[] = [
  {
    id: "characters",
    name: "Check Player Characters",
    description:
      "Check if characters are named correctly and have allowed ascendancies",
  },
  {
    id: "invites",
    name: "Private League Invites",
    description:
      "Process and accept private league invites for registered players",
  },
  {
    id: "stash",
    name: "Guild Stash Logs",
    description: "Fetch guild stash changes and sync to BPL backend",
  },
];

export const CREDENTIALS: Record<
  CredentialKind,
  {
    name: string;
    description: string;
    link: { url: string; label: string };
    warning?: string;
    steps?: string[];
  }
> = {
  bpl_token: {
    name: "BPL Token",
    description: "BPL API token for authentication",
    link: { url: "https://bpl-poe.com/token", label: "Open bpl-poe.com/token" },
    steps: [
      "Open the page below in your browser and log in if asked",
      "Click Copy",
      "Paste the token here",
    ],
  },
  poe_session: {
    name: "PoE Session ID",
    description: "Path of Exile session ID from browser",
    link: {
      url: "https://www.pathofexile.com/my-account",
      label: "Open pathofexile.com and log in",
    },
    warning:
      "This is extremely sensitive data. Please check the code to see how it's used.",
  },
};
