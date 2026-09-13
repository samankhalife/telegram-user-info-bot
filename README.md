# Telegram User Info Bot

A Telegram bot written in Go that explains and displays the user, chat, message, media, forwarding, contact, and location metadata Telegram exposes through the Bot API. It also keeps an audit record of every supported incoming request and every corresponding response in SQLite for 30 days.

## What the bot does

For each supported Telegram update, the bot follows this flow:

1. Receives the update from Telegram through long polling.
2. Stores the complete update JSON and useful searchable identifiers in SQLite.
3. Reads only fields Telegram included in that update.
4. Formats those fields into a human-readable HTML report.
5. Sends one or more response messages back to the same chat.
6. Stores every response attempt, including its text, keyboard markup, Telegram message ID, delivery status, and error when sending fails.
7. Deletes interactions and linked responses after 30 days.

The bot supports ordinary messages, edited messages, channel posts, and edited channel posts. Reports can include:

- Telegram update, message, user, and chat IDs
- First name, last name, username, language, and bot status
- Chat type, title, username, description, and related metadata
- Message text, caption, timestamps, reply and forward metadata
- Photo, document, video, audio, voice, sticker, and animation metadata
- Poll, dice, membership, and selected service-message metadata
- A contact or location only when it is present in the update

Long reports are split into Telegram-safe message sizes. User-controlled values are HTML-escaped before being sent.

## Telegram API limitations

This bot **cannot discover hidden or private information**. It cannot retrieve:

- Telegram passwords or login codes
- IP addresses
- Email addresses
- Hidden phone numbers
- A user's current location without explicit sharing
- Private messages from other conversations
- Arbitrary message history
- Information Telegram did not include in the update

A phone number or location becomes available only when the user explicitly presses Telegram's share-contact or share-location button in a private chat. Forwarded-message privacy settings may also hide the original sender.

## Data storage and privacy

The bot intentionally stores full interactions. The SQLite database can contain personal data, including:

- Complete incoming Telegram update JSON
- Message and caption text
- User, chat, message, and update IDs
- Names and usernames
- Contact phone number and vCard when shared
- Latitude and longitude when shared
- Media file IDs and metadata
- Complete response text and keyboard markup
- Delivery status, Telegram response message ID, and sanitized send errors
- Receive and send timestamps

Records older than `RETENTION_DAYS` are automatically deleted at startup and once every 24 hours. The default retention is 30 days. Deleting an interaction also deletes all linked responses.

The SQLite database is not encrypted by the application. Restrict filesystem and Docker-volume access, protect backups, and comply with applicable privacy laws and consent requirements. Do not publish or commit the database.

## Requirements

- Go 1.23 or newer, or Docker with Docker Compose
- A Telegram bot token created through [@BotFather](https://t.me/BotFather)
- Network access to `https://api.telegram.org`

## Configuration

Copy the example file:

```bash
cp .env.example .env
```

Available variables:

```env
BOT_TOKEN=123456789:replace-with-your-real-token
LOG_LEVEL=info
DATABASE_PATH=./data/bot.db
RETENTION_DAYS=30
```

- `BOT_TOKEN` is required.
- `LOG_LEVEL` supports `debug`, `info`, `warn`, and `error`.
- `DATABASE_PATH` defaults to `./data/bot.db`.
- `RETENTION_DAYS` must be a positive integer and defaults to `30`.

The application does not automatically load `.env` when launched directly with Go.

## Run locally

Load `.env` into the current shell and start the bot:

```bash
set -a
source .env
set +a
go run ./cmd/bot
```

The local database is created at `./data/bot.db` by default. The `data/` directory and SQLite sidecar files are ignored by Git.

Stop the process with `Ctrl+C`. The bot handles `SIGINT` and `SIGTERM`, stops long polling, and closes SQLite cleanly.

## Run with Docker in the background

Build and start the bot in detached mode:

```bash
docker compose up -d --build
```

Docker Compose overrides `DATABASE_PATH` with `/data/bot.db` and mounts a named volume called `bot-data`. The database therefore survives container restarts, rebuilds, and replacement.

Useful commands:

```bash
# Show container status
docker compose ps

# Follow logs
docker compose logs -f telegram-user-info-bot

# Restart the bot
docker compose restart telegram-user-info-bot

# Stop and remove the container without deleting stored data
docker compose down

# List the persistent volume
docker volume ls | grep bot-data
```

The service uses `restart: unless-stopped`, so it starts again after a process crash or Docker/host restart unless explicitly stopped.

To permanently delete the container **and all stored interaction data**, run:

```bash
docker compose down -v
```

This operation is destructive and cannot be undone unless you have a backup.

## Commands

- `/start` — explains usage, storage, and privacy; displays optional contact/location sharing buttons
- `/help` — displays the same usage information
- `/info` — displays metadata for the command message
- Any ordinary message — displays metadata Telegram supplied with that message

## Database structure

The `interactions` table stores one unique row per Telegram `update_id`. This prevents polling retries from creating duplicate request rows. The `responses` table stores one or more response attempts linked to the interaction.

SQLite runs with foreign keys, WAL mode, a busy timeout, and a single application database connection. This design is intended for one running bot instance. Use a server database such as PostgreSQL before horizontally scaling to multiple bot replicas.

You can inspect a local database with the SQLite CLI:

```bash
sqlite3 ./data/bot.db
```