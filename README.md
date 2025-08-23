# Tools for BPL Admins

## Requirements

We need the following input Parameters:

- PoE Session ID
  - This is EXTREMELY sensitive data. In fact it's already kind of sketchy that we are asking you to give it to this tool as an input. Please check out the code to see how we are actually using it.
- BPL JWT
  - This is your token to authenticate yourself against the BPL Backend
- Guild ID
  - This is the internal ID of your guild

# How to get the parameters:

## PoE Session ID

- Navigate to https://www.pathofexile.com/
- Make sure you are logged in
- Press F12 to open dev tools
- Go to Application (Chromium) / Storage (Firefox)
- Go to Cookies for https://www.pathofexile.com/
- Select the value for POESESSID

## BPL JWT

- Navigate to https://bpl-poe.com/
- Make sure you are logged in
- Press F12 to open dev tools
- Go to Application (Chromium) / Storage (Firefox)
- Go to Local storage for https://bpl-poe.com/
- Select value for auth

## Guild ID

- Navigate to https://www.pathofexile.com/my-guild
- Click on "Stash History"
- Your Browser URL should now look like this: https://www.pathofexile.com/guild/profile/408208/stash-history
- The number between profile and stash-history is your guild id

## Private League ID

- Navigate to https://www.pathofexile.com/private-leagues
- Click view on your league
- It will be named "YourLeagueName (PL12345)
- The number after PL is your league id

## Usage

### For End Users

1. Download the latest release from the [Releases page](https://github.com/BPL-v2/tools/releases)
2. Extract the zip file for your platform - this will create a `bpl-tools` folder
3. Navigate to the `bpl-tools` folder
4. Run the tool: `./bpl-tools` (Linux/macOS) or `bpl-tools.exe` (Windows)
5. Configuration will be automatically saved to `bpl-config.txt` in the same folder

**Note:** No setup required! The application will automatically prompt you for any missing environment variables when you try to use a feature that requires them, and it will save your inputs to the `bpl-config.txt` file for future use.

### Interactive Menu

The tool provides an easy-to-use interactive menu:

- Use ↑/↓ arrow keys to navigate
- Press Enter to select
- Choose your tool, then choose run mode (once or continuous)
- When using a feature for the first time, you'll be prompted to enter required credentials
- The application will offer to show step-by-step instructions for obtaining each credential
- Sensitive values (tokens, session IDs) are hidden while typing for security

### Environment Variables

The application uses environment variables stored in a `bpl-config.txt` file for configuration. The tool will automatically:

- Prompt you for any missing required environment variables when you first use a feature
- Offer to show detailed instructions on how to obtain each environment variable
- **Detect invalid credentials** and offer to re-enter them when API calls fail
- Save your inputs to the `bpl-config.txt` file for future use
- Use hidden input for sensitive values (tokens, session IDs)
- Load existing values from the `bpl-config.txt` file on startup

Required variables by feature:

- **Check Player Characters**: No environment variables required
- **Handle Private League Invites**: `BPL_TOKEN`, `POESESSID`
- **Guild Stash Monitor**: `BPL_TOKEN`, `POESESSID`

## Development

### Building from Source

```bash
git clone https://github.com/BPL-v2/tools.git
cd tools
go build -o bpl-tools .
```

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

- Build binaries for all platforms (Linux, Windows, macOS Intel/ARM)
- Create simplified zip packages with just the executable
- Create a GitHub release with download links
- Generate release notes with quick start instructions
