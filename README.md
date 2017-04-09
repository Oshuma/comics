# Comics

Tablet-first comic reader.

## Setup

Local development is done with [Docker Compose](https://docs.docker.com/compose/) and uses the [ruby](https://hub.docker.com/_/ruby/) and [postgres](https://hub.docker.com/_/postgres/) repos.
Make sure you have Docker Compose installed, then run:

```
$ docker-compose up
```

## Deploy

Any Unix-like server running Ruby 2+ and Postgres 9.5+ should work.

| Requirements | | |
| ------------ |-|-|
| imagemagick | `apt-get install imagemagick` | |
| unrar | `apt-get install unrar` | (debian note: this must be the 'non-free' package) |
| unzip | `apt-get install unzip` | |
