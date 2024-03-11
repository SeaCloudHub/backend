# Template for go team

## Development

### Init local environment

1. Copy file `.env.example` and rename to `.env`

2. Update env vars to fit your local

3. Start local services

   ```shell
   make db
   ```

4. Run the migration

   ```shell
    make migrate
   ```

5. Create admin account

   ```shell
   make seed
   ```

6. Run the server

   ```shell
   make run
   ```

7. Unit test
   ```shell
   make test
   ```

### Linting

```shell
make lint
```

### Create new migration file

```shell
sql-migrate new -env="development" create-users-table
```

- Result: `Created migration migrations/20230908204301-create-user-table.sql`

### Swagger

- Access to `{host}/swagger/index.html`

- Install swag CLI

```shell
go install github.com/swaggo/swag/cmd/swag@latest
```

- Generate swagger docs

```shell
make swagger
```