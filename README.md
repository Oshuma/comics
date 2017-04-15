# Comics

Tablet-first comic reader.

## Setup

Local development is done with [Docker Compose](https://docs.docker.com/compose/) and uses the [ruby:2.3-alpine](https://hub.docker.com/r/library/ruby/tags/2.3-alpine/) and [postgres:alpine](https://hub.docker.com/r/library/postgres/tags/alpine/) repos.
Make sure you have Docker Compose installed, then run:

```
$ docker-compose up
```

Once both containers are running (`web` and `db`), you'll need to create the databases and run migrations:

```
$ docker-compose run web rake db:create
$ docker-compose run web rake db:migrate
```

There's a wrapper script `web.sh` that can be used to run commands in the `web` container:

```
$ ./web.sh rake routes
```

## Deploy

Any Unix-like server running Ruby 2+ and Postgres 9.5+ should work.

| Requirements | | |
| ------------ |-|-|
| imagemagick | `apt-get install imagemagick` | |
| unrar | `apt-get install unrar` | (debian note: this must be the 'non-free' package) |
| unzip | `apt-get install unzip` | |
