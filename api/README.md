# Hortus API
The API of Hortus web server, an online plant manager.

## Getting started
### Run the server
1. Clone the repo

```bash
git clone https://github.com/mgmu/hortus-api
```

2. `cd` into the new directory

```bash
cd hortus-api
```

3. Build the binary

```bash
go build
```

4. Export the database connection URL. Hortus API uses PostgreSQL.

```bash
export HORTUS_DB_URL=postgres://<user>:<password>@<ip address>:<port>/<database name>
```

> Note that you have to create a PostgreSQL user, a database, the
> `hortus_schema` schema in that database and run the `init_hortus_db.sql`
> script beforehand.

5. Run the server. The server listens for HTTP connections at port `8080`.

```bash
./hortus-api
```
