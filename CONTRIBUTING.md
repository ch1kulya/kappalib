### Any help is welcome here

You can create issues here. Please comment on them if you encounter a problem that has already been described. I also welcome any pull requests, but please take responsibility for what you submit.

> [!WARNING]
> The primary development branch is `dev`. Please use it in your forks and submit pull requests against this branch.

### How to set up a dev environment

To get started, you will need Go version `1.27.0` and Docker with Docker Compose.

You will also need to install a hot reload utility:

```shell
go install github.com/air-verse/air@v1.67.4
```

And install the templating engine for Go:

```shell
go install github.com/a-h/templ/cmd/templ@v0.3.1020
```

Fill in `.env` and `.env.dev` based on their `.example` versions. You can leave the `DEV_DATABASE_URL` value as is if you plan to use a mock database. And `AIR_PROXY_HASH` should be left as is if you have installed the correct version of air.

### How to run it

Start the containers for the mock database:

```shell
docker compose -f docker-compose.dev.yml up -d
```

Populate it with data using this command:

```shell
go run cmd/seed/main.go
```

And finally, launch the server:

```shell
air
```

If you forget something during the process, you will receive an error.

### Code formatting

The project uses [dprint](https://dprint.dev) to format source code across languages and configurations.

[Install dprint](https://dprint.dev/install/) and additional formatting utilities required by `dprint.json`:

- `shfmt` for shell scripts:

```shell
go install mvdan.cc/sh/v3/cmd/shfmt@latest
```

- `templ` for templates (if not already installed):

```shell
go install github.com/a-h/templ/cmd/templ@v0.3.1020
```

- `pg_format` for SQL (requires Perl):

```shell
git clone --depth 1 https://github.com/darold/pgFormatter.git tools/pg_format
```

To check formatting without modifying files:

```shell
dprint check
```

To format files:

```shell
dprint fmt
```
