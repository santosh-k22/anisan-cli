<p align="center">
  <strong><h1 align="center">AniSan ⚡</h1></strong>
</p>

<p align="center">
  <em>A modular anime CLI, forged in Go — where the anime CLI ecosystem converges.</em>
</p>

<p align="center">
  <a href="https://github.com/santosh-k22/anisan-cli/blob/main/LICENSE">
    <img src="https://img.shields.io/badge/license-AGPL--3.0-blue.svg" alt="License">
  </a>
  <a href="https://go.dev">
    <img src="https://img.shields.io/badge/Go-1.25-00ADD8?logo=go&logoColor=white" alt="Go Version">
  </a>
  <a href="https://github.com/santosh-k22/anisan-cli/releases">
    <img src="https://img.shields.io/badge/platform-linux%20%7C%20macos%20%7C%20windows-lightgrey" alt="Platform">
  </a>
</p>

---

AniSan is a modular, performant CLI for **browsing, streaming, and tracking anime**. It provides a flexible Lua-based scraping system and a fast TUI experience.

## 🎯 Features

| Feature | Description |
|---|---|
| 🔌 **Lua-Powered Scrapers** | Easily extensible sources (AllAnime ships by default). |
| 🖼️ **Terminal Img Renderer**| High-fidelity Anime Cover Art rendered dynamically during TUI search/history via TrueColor ANSI blocking. |
| 🎬 **MPV Integration** | Background IPC control, auto-resume, visual chapter markers, and performance-optimized event polling. |
| 📡 **Deterministic Sync**| Forensic synchronization via native event triggers (EOF heist), ensuring reliable tracker updates even on instant player exits. |
| 📊 **Rich History Model** | Visually tracks watch progress, statuses, aggregated scores, and associated genres locally. |
| ⚡ **7-Day Caching** | Instantaneous TUI loading powered by a native Go metadata cache. |
| ⏩ **Auto-Skip Intro** | Integration with [AniSkip](https://api.aniskip.com) to skip OP/ED. |
| 🛡️ **Atomic Guarding** | Shared `sync/atomic` guards prevent double-synchronization or race conditions between background watchers and TUI handlers. |
| 🎨 **TUI** | Beautiful "Catppuccin" themed UI with fluid wrap-around navigation and native autocomplete. |
| 🍏 **IINA Support** | Native macOS player integration with full feature parity. |

## 📦 Installation

### From Source (Recommended)

```bash
git clone https://github.com/santosh-k22/anisan-cli.git
cd anisan-cli
go install .
```

### Dependencies

- **Go** 1.25+ (to build)
- **mpv** (required for playback)
    - macOS: `brew install mpv` (or use **IINA**)
    - Linux: `sudo apt install mpv`
    - Windows: `scoop install mpv`

### Using IINA (macOS only)
To use IINA instead of MPV on macOS, update your configuration:
```bash
anisan config set player.default iina
```

## ⚡ Usage

### Interactive Mode (TUI)
Simply run the command to enter the interactive interface:
```bash
anisan
```

### Quick CLI Mode (Inline or Headless)
Search, stream, or sync progress without entering the TUI:

```bash
# Search and select
anisan inline -q "Attack on Titan"

# Auto-select first result and play first episode
anisan inline -q "Bleach" --source allanime --anime first --episode first

# Continue watching from history
anisan inline --continue

# Mark an episode as watched headlessly (syncs to configured tracker)
anisan mark -q "Jujutsu Kaisen" -e 12
```

## ⚙️ Configuration

AniSan is incredibly customizable. You can control the app through the CLI configuration command, or by directly editing the active config file.

### Command-Line Configuration
You can read or overwrite ANY built-in config directly from your shell by passing key-value pairs.
```bash
# Get a value
anisan config get player.aniskip

# Set a value
anisan config set player.aniskip false

# Change Tracker Backend (anilist or mal)
anisan config set tracker.backend mal

# Reset a value to default
anisan config reset --key player.aniskip
```
*Tip: Run `anisan config get` without any keys to view your entire active configuration tree.*

## 🔗 Syncing with MyAnimeList & AniList
AniSan natively supports background episode synchronization to both MyAnimeList and AniList.
You can configure your preferred primary tracker using the `tracker.backend` configuration key (defaults to `anilist`).

To authenticate the trackers, use the built-in oauth agents:
```bash
# Authorize AniList
anisan anilist auth

# Authorize MyAnimeList
anisan mal auth
```
*Note: The interactive TUI will proactively check your authentication status before playback. It utilizes a shared atomic synchronization guard to ensure that your progress is updated exactly once, either by the background IPC watcher or the exit handler, preventing redundant tracker requests.*

### Key Configuration Options

| Key | Default | Description |
|---|---|---|
| `tracker.backend` | `anilist` | Active media synchronization backend (`anilist` or `mal`). |
| `tracker.enable` | `false` | Enable remote media tracker integration. |
| `tracker.auto_link` | `true` | Prompt to safely auto-link to Tracker when selecting an anime. |
| `player.default` | `mpv` | Media player to use (e.g., `mpv`, `iina`). |
| `player.aniskip` | `true` | Enable/Disable auto-skipping intros via AniSkip. |
| `player.completion_percentage` | `80` | Percentage required to automatically mark an episode as watched (1-100). |
| `tui.show_urls` | `true` | Show URLs in the interactive list view. |
| `icons.variant` | `plain` | Icon set (`emoji`, `nerd`, `plain`, `kaomoji`). |

*(See `anisan config info` for a full list of options)*

### The `anisan.toml` Config File
For power users, AniSan naturally stores all configurations in a unified `toml` file. Changes made here apply instantaneously to the CLI.

**Location**:
- **macOS/Linux**: `~/.config/anisan/anisan.toml` (or `~/Library/Application Support/anisan/anisan.toml`)
*(You can override this by exporting `ANISAN_CONFIG_PATH=/custom/path` in your `.zshrc`)*

**Example `anisan.toml`**:
```toml
[player]
default = "mpv"  # Options: mpv, iina
aniskip = true   # Auto-skip anime intros/outros
completion_percentage = 80  # Mark watched at 80%

[tui]
item_spacing = 1
read_on_enter = true
show_urls = true
reverse_episodes = false
```

### Automatic Intro Skipping (AniSkip)
AniSan natively integrates with **AniSkip** via an internal Go implementation—no external Python or Lua scripts required.

When you play an episode, AniSan silently queries `api.aniskip.com` for the exact timestamps of the Opening (OP) and Ending (ED). It then monitors a hidden Unix socket and dispatches lightning-fast `seek` commands to `mpv` to seamlessly bypass them.

*(Note: AniSkip currently requires `mpv`. Native players like `iina` operate independently of the socket).*

```bash
# To disable AniSkip:
anisan config set player.aniskip false
```

## 📡 Integrations

### MyAnimeList (MAL)
Authenticate to sync your watch progress automatically.

1. **Authenticate**:
   First, register an API application on your [MyAnimeList API Panel](https://myanimelist.net/apiconfig).
   Copy the generated Client ID and set it as an AniSan config:
   ```bash
   anisan config set tracker.mal.client_id "YOUR_MAL_CLIENT_ID"
   ```
   Then trigger the OAuth2 flow:
   ```bash
   anisan mal auth
   ```
   This will open your browser. Login and approve the app.

2. **Usage**:
   - In the episode list, press `t` to search and link the anime entry dynamically based on your active `tracker.backend` configuration.
   - Once linked, progress will sync automatically when you reach ~80% of an episode, or instantly upon reaching the end (EOF) via native IPC events.

### Anilist
1. **Authenticate**:
   ```bash
   anisan anilist auth
   ```
   This will open your browser, request access for AniSan, and safely store the token.
2. **Usage**:
   - In the episode list, press `t` to search and link the anime to an Anilist entry manually (if `anilist` is set as the active backend).

### Progress Syncing & Manual Overrides
AniSan syncs your watch history directly to MyAnimeList (MAL) or Anilist automatically once you surpass the `completion_percentage` (default 80%) or when the video completes (EOF).

Sometimes, a scraped Anime title doesn't perfectly match the MAL/Anilist database (e.g., alternate English names). If auto-sync fails to recognize the show:
1. Open the Anime's **Episodes Menu**.
2. Press **`I`** to open the **Manual ID Override** modal.
3. Type in the official MAL or Anilist ID for the show.
4. Hit Enter to instantly fetch and permanently cache the official metadata. All future episodes will sync perfectly.

## ⌨️ Keybindings & TUI Navigation

AniSan's TUI is designed for speed. **Wrap-Around Scrolling** is enabled globally—pressing `Up` on the first item jumps to the bottom, and vice versa.

**Global Navigation**
- `↑` / `k`: Up (Wraps around)
- `↓` / `j`: Down (Wraps around)
- `←` / `h`: Previous page
- `→` / `l`: Next page
- `/` : Activate fuzzy filtering
- `tab` : Accept active search suggestion
- `g`: Jump to Top
- `G`: Jump to Bottom
- `q` / `ctrl+c`: Quit
- `esc`: Back / Cancel
- `?`: Toggle help menu

**Interactive Overrides & States**
- `S`: Change source / Save as Default (in Source Selection)
- `d`: Delete entry (in History State)

**Episode List Actions**
- `enter`: Play selected episode
- `space` : Select a single episode (useful for batching)
- `tab` / `ctrl+a` / `*` : Select all episodes in list
- `backspace` : Clear all active selections
- `t`: Search/Link to your currently configured active Tracker dynamically
- `I`: **Manual ID Override** (Input MAL/Anilist ID directly)
- `v`: Select all episodes strictly by Volume
- `o`: Open episode source URL in browser

**Post-Watch Menu & Player Binds**
*(Player keybindings available natively inside the CLI when MPV is hosted)*
- `space`: Pause/Resume active playback
- `n`: Skip to Next chronological episode
- `p`: Revert to Previous chronological episode
- `r`: Replay the current episode
- **Menu Options** *(Appears when episode finishes or player is closed)*: Next, Replay, Previous, Back to Episodes
## 🛠️ Utility Commands

Beyond traditional TUI interaction, AniSan ships with an array of built-in CLI-native utility tools for power-users, developers, and advanced scripters.

| Command | Action |
|---|---|
| `anisan sources list` | List all discovered built-in and user-installed Lua providers |
| `anisan sources gen` | Scaffolds a complete Lua template to develop a custom provider |
| `anisan where` | Print localized filesystem locations (`config`, `cache`, `sources`, etc) |
| `anisan env` | List all available framework environment variables |
| `anisan clear` | System utility to wipe tracking caches, queries, or corrupt histories |
| `anisan mini` | Launch an experimental, ultra-lightweight minimalist search CLI |
| `anisan run` | Execute and debug a local Lua scraper file headless via Go-Lua |
| `anisan version` | Output exhaustive build metrics and metadata |

## 🤝 Acknowledgments & Inspiration

This project is an **original work**, written from scratch in Go. No code was copied from any of the projects below. However, AniSan draws significant design inspiration from this thriving anime CLI ecosystem. Big thanks to the creators of these fantastic tools for paving the way and providing ideas for features, minimal workflows, and scraping methodologies:

- [mangal](https://github.com/metafates/mangal) — A massive inspiration for the core architecture. AniSan heavily adopts Mangal's brilliant paradigm of combining a compiled Go engine with embedded Lua scrapers, allowing for incredible extensibility without sacrificing performance. (MIT)
- [ani-cli](https://github.com/pystardust/ani-cli) — The godfather of anime CLIs. AniSan was deeply inspired by its baseline Unix philosophy, straightforward user experience, and dedicated community. (GPL-3.0)
- [animdl](https://github.com/justfoolingaround/animdl) — Inspired our high-performance stream extraction concepts and robust scraping capabilities. (GPL-3.0)
- [ani-skip](https://github.com/KilDesu/ani-skip) — The amazing API powering our automated OP/ED skip logic. (MIT)
- [curd](https://github.com/Wraient/curd) — Inspired our sync methodologies, Anilist integration patterns, and MPV state tracking mechanisms. (GPL-3.0)
- [GoAnime](https://github.com/alvarorichard/GoAnime) — A fantastic reference for TUI responsiveness and effective BubbleTea implementations. (MIT)
- [jerry](https://github.com/justchokingaround/jerry) — A great example of functional minimalism and workflow efficiency. (GPL-3.0)
- [anipy-cli](https://github.com/sdaqo/anipy-cli) — For architectural decoupling ideas. (GPL-3.0)
- [viu](https://github.com/viu-media/viu) — For modularity and advanced HTTP handling ideas. (Unlicense)

## 📄 License

This project is licensed under the [GNU Affero General Public License v3.0](LICENSE).

You are free to use, modify, and distribute this software under the terms of the AGPL-3.0. If you modify the source and provide it as a network service, you must make your modified source code available. See the [LICENSE](LICENSE) file for the full legal text.
