# beeper-ingestor

A Matrix message search service that uses [gomuks](https://github.com/tulir/gomuks) internals to save incoming events and provides a REST endpoint for searching messages.

> 🤖 **Note**: This repository, including this README, was primarily generated using Large Language Models (LLMs). Code has been tested and works as intended though!

## Setup

### Prerequisites

- Go
- SQLite3
- Environment variables:
  - `GOMUKS_ROOT`: Base directory for gomuks data (required)
  - `ACCESS_LIST`: Authentication credentials in format `user:hashedpass|user2:hashedpass2` (required)

### `GOMUKS_ROOT`

The service expects the following directory structure under `GOMUKS_ROOT`:

```
GOMUKS_ROOT/
├── cache/
├── config/
│   └── config.yaml  # Required gomuks configuration
├── data/
└── logs/
```

You can setup the account using gomuks itself and then switch to running this program.

### Building

```bash
./build.sh
```

### API authentication

The service uses Basic Authentication with SHA-256 hashed passwords. Passwords must be hashed and base64 encoded before being added to the `ACCESS_LIST` environment variable.

Use the provided `generate-password.py` script to generate hashed passwords:

```bash
python3 generate-password.py <your-password>
```

Then set the `ACCESS_LIST` environment variable with username:hashedpassword pairs:

```bash
export ACCESS_LIST="user1:hashedpass1|user2:hashedpass2"
```

## API Reference

### Search Messages

`GET /search-messages`

Search for messages with various filters. Requires Basic Authentication.

#### Query Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| room_id | string | Filter messages by room ID |
| sender | string | Filter messages by sender. Will automatically add @ prefix if missing. Must include domain (e.g. @user:domain.com) |
| before | integer | Filter messages before this timestamp (milliseconds since epoch) |
| after | integer | Filter messages after this timestamp (milliseconds since epoch) |
| limit | integer | Maximum number of messages to return (default: 100, max: 1000) |
| cursor | string | Pagination cursor (event rowid) |
| direction | string | Pagination direction, must be "before" or "after" when cursor is provided |

#### Response Format

```json
{
  "items": [
    {
      "id": "string",
      "timestamp": "number",
      "senderID": "string",
      "text": "string",
      "url": "string",
      "roomInfo": {
        "id": "string",
        "name": "string",
        "url": "string"
      }
    }
  ],
  "has_more": "boolean",
  "oldest_cursor": "string",
  "newest_cursor": "string"
}
```

#### Example Request

```bash
curl -u username:password 'http://localhost:8080/search-messages?room_id=!roomid:domain.com&limit=10'
```
