# Tools for BPL Admins

A desktop app built with [Tauri](https://tauri.app/), using the same React + Tailwind + daisyUI stack as the BPL frontend.

## Requirements

We need the following input Parameters:

- PoE Session ID
  - This is EXTREMELY sensitive data. In fact it's already kind of sketchy that we are asking you to give it to this tool as an input. Please check out the code to see how we are actually using it.
- BPL JWT
  - This is your token to authenticate yourself against the BPL Backend

The guild ID and private league ID are detected automatically.

# How to get the parameters:

The app walks you through both of these, including where to click in your browser.

## PoE Session ID

- Log in on https://www.pathofexile.com/my-account
- Press F12 to open dev tools
- Go to Application (Chromium) / Storage (Firefox)
- Go to Cookies for https://www.pathofexile.com/
- Select the value for POESESSID

## BPL JWT

- Open https://bpl-poe.com/token and log in if asked
- Click Copy

## Usage

### For End Users

1. Download the installer for your platform from the [Releases page](https://github.com/BPL-v2/tools/releases)
2. Install and start BPL Tools
3. Pick a tool in the sidebar, then choose **Run Once** or **Run Continuously** (every 5 minutes)

**Note:** No setup required! When a tool needs a credential that is missing or was rejected, the app asks for it, shows step-by-step instructions, and retries the run. You can also manage both values on the **Credentials** page.

### Credentials

Credentials are stored in your system keychain: Keychain on macOS, Credential Manager on Windows and the Secret Service (KWallet, GNOME Keyring) on Linux. They stay in the Rust backend, the UI can only see whether they are set.

An existing `bpl-config.txt` from an older version is moved into the keychain on launch and deleted afterwards. The app looks for it in its config directory, next to the app and in the directory it was started from.

If no keychain is available, credentials are kept unencrypted in `bpl-config.txt` in the app's config directory, readable only by your user on Linux and macOS. The Credentials page warns you when this happens, and the values move to the keychain once one is available.

Required variables by feature:

- **Check Player Characters**: No credentials required
- **Private League Invites**: `BPL_TOKEN`, `POESESSID`
- **Guild Stash Logs**: `BPL_TOKEN`, `POESESSID`

## Development

You need Node.js, a Rust toolchain and the [Tauri system dependencies](https://v2.tauri.app/start/prerequisites/).

```bash
git clone https://github.com/BPL-v2/tools.git
cd tools
npm install
npm run tauri dev    # run the app with hot reload
npm run tauri build  # build installers for the current platform
```

Tests:

```bash
npm test
cargo test --manifest-path src-tauri/Cargo.toml
# also check the models against the production API
cargo test --manifest-path src-tauri/Cargo.toml -- --ignored
```

The UI lives in `src/`, the tools and all network access in `src-tauri/src/`. Tools report back to the UI through `tool-event` events, see `src-tauri/src/jobs.rs`.

### Creating a Release

To create a new release (for maintainers):

```bash
# Make sure all changes are committed
git add .
git commit -m "Your changes"
git push

# Create and push a release tag
./release.sh v1.0.0
```

This will automatically:

- Build installers for all platforms (Linux, Windows, macOS Intel/ARM)
- Create a GitHub release with download links
- Generate release notes with quick start instructions
