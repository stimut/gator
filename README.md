# Gator

Gator is a RSS aggregator written as part of a programming exercise.

## Requirements

* PostgreSQL - used as the storage layer for the RSS feeds and posts
* Go - no pre-built executables are distributed, so it must be compiled from source

## Installation

Using the Go CLI simply run:

```bash
go install github.com/stimut/gator
```

## Configuration

The configuration is read from `$HOME/.gatorconfig.json`. A minimal config would be:

```json
{
  "db_url": "<url-of-postgres-db>"
}
```

## Usage

```bash
gator register <user>       # create a new user
gator login <user>          # login/switch as a user
gator addfeed <name> <url>  # registers a new RSS feed
gator follow <url>          # follow a feed that a different user already registered
gator agg [refresh-time]    # aggregate feeds, checking a feed every [refresh-time] (does not exit)
gater browse [num]          # print [num] most recent posts for current user
```

## Contributing

Honestly, this is a programming exercise and not a real project. It will be abandoned as soon as
it is uploaded.