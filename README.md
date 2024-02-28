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

5. Run the server

   ```shell
   make run
   ```

6. Unit test
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
