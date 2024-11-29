# GoMaCal

> [!NOTE]
> "Go get ma calendar"
> "Google, ma calendar please"

A simple command-line Google calendar fetcher that shows your upcoming Google Calendar meetings with meeting links. Perfect for automation scripts and system notifications.

## Features

- Shows upcoming calendar events within a specified time window
- Extracts meeting links (Google Meet, Zoom, Teams, etc.)
- Formats output in a parseable way for scripts
- Supports duration-based queries (e.g., "next 5 minutes")
- Remembers authentication (only need to authorize once)

## Installation

```bash
go install github.com/yourusername/gomacal@latest
```

Or clone and build:
```bash
git clone https://github.com/yourusername/gomacal
cd gomacal
go build
```

## Setup

Before using GoMaCal, you need to set up Google Calendar API access. Don't worry, it's free!

1. Go to [Google Cloud Console](https://console.cloud.google.com/)
2. Create a new project (or select an existing one)
3. Enable the Google Calendar API:
   - Go to "APIs & Services" > "Library"
   - Search for "Google Calendar API"
   - Click "Enable"
4. Create credentials:
   - Go to "APIs & Services" > "Credentials"
   - Click "Create Credentials" > "OAuth client ID"
   - Choose "Desktop application" as the application type
   - Give it a name (e.g., "GomaCal")
   - Click "Create"
5. Download the credentials:
   - Find your newly created OAuth 2.0 Client ID
   - Click the download button (JSON)
   - Rename the downloaded file to `credentials.json`
   - Place it in the same directory as the `gomacal` binary

## Usage

First run will require Google authentication:
```bash
./gomacal
```
This will open your browser for authorization. You only need to do this once.

Check upcoming events:
```bash
# Show all today's events
./gomacal

# Show events in next 5 minutes
./gomacal --next 5m

# Show events in next 2 hours
./gomacal --next 2h
```

## Output Format

Events are output in the following format:
```
Meeting Title § Time Info § Meeting Link
```

Example:
```
Team Standup § starts in 5m § https://meet.google.com/abc-defg-hij
```

## Using with Scripts

The output format makes it easy to parse in scripts. Here's an example in Lua:
```lua
local function parse_event(event_string)
    local title, time_info, meeting_link = event_string:match("(.+) § (.+) § (.*)")
    return {
        title = title,
        time_info = time_info,
        meeting_link = meeting_link ~= "" and meeting_link or nil
    }
end
```

## Note on Google Cloud Console

The Google Cloud Console might look intimidating, but for this usage:
- It's completely free (Calendar API is free for personal use)
- You only need to set it up once
- You're only accessing your own calendar
- No billing account required

## License

MIT
