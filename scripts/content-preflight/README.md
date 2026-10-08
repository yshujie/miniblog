# Content deployment preflight

The production image contains `/app/content-preflight`. Run it after pulling the
new image and before replacing the running service:

```sh
docker compose -f docker-compose.yml -f docker-compose.prod.yml run --rm --no-deps --entrypoint /app/content-preflight miniblog-backend
```

A nonzero exit must stop deployment. The tool never migrates, creates tables,
repairs a dirty version, backfills article identities, or requests Notion. It
uses a read-only transaction and reads only migration metadata,
`information_schema`, and the existence of the sync control singleton.

Connection settings come only from the environment. `MINIBLOG_DATABASE_HOST`,
`PORT`, `USERNAME`, `PASSWORD`, and `DBNAME` take precedence; `DATABASE` is also
accepted. `MYSQL_HOST`, `PORT`, `USERNAME` (or `USER`), `PASSWORD`, and `DATABASE`
(or `DBNAME`) are fallbacks. Only the port defaults to 3306. Credential flags and
`MYSQL_DSN` are not accepted; no configuration file is loaded. Driver errors and
connection settings are never printed. The complete check has a 30-second
limit, with 10-second connection and socket timeouts.

Checks require one clean `schema_migrations` row with version at least 6, all
content and sync columns, InnoDB tables, full-column identity constraints,
binary identity collations, `article.tags_json` as LONGTEXT, and the initialized
sync control row. The source unique index must have its expected
`uq_article_source_key` name because the registration gate also checks it.
A higher version still requires these objects. Index/table names in failures
are fixed schema identifiers, never article contents or credential values.

This validates deployment structure only. Approved historical data audit,
backfill and migration must run separately; a successful check does not enable
Notion sync or confirm real Notion token/share access.

Local verification (no database or credentials required):

```sh
go test -race ./scripts/content-preflight
go vet ./scripts/content-preflight
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /tmp/content-preflight ./scripts/content-preflight
```

Tests exercise incomplete/dirty migration gates, missing fields/tables,
prefixed/non-unique/wrong identity indexes, environment selection, error
redaction, and SELECT-only inspection under a read-only transaction using an
in-process database driver. Real MySQL and the built Docker image require
separate deployment verification.
