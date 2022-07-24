# Comics

Web based, tablet-first comic reader.

![Groups](public/screenshots/001.png)
![Group](public/screenshots/002.png)
![Page](public/screenshots/003.png)

## Development Setup

Local development is done with [Docker Compose](https://docs.docker.com/compose/) and uses the [ruby:alpine](https://hub.docker.com/_/ruby) and [postgres:alpine](https://hub.docker.com/_/postgres) repos.
Make sure you have Docker Compose installed, then run:

```
$ docker-compose up
```

Once both containers are running (`web` and `db`), run the setup helper:

```
$ docker-compose run --rm web ./bin/setup
```

There's a wrapper script `web.sh` that can be used to run commands in the `web` container:

```
$ ./bin/web.sh rake routes
```

## Use

After the containers are running and the app is setup, hit [http://localhost:3000](http://localhost:3000) in your browser and you should be redirected to the initial User setup.
This will setup an admin account, which you can later use to add more users, etc.
