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

## Docs
Summary table of API endpoints:

| Endpoint | Description | Allowed methods |
| :-:      | :-          | :-              |
| `/plants/` | Get the list of plant names and ids. | `HEAD`, `GET`|
| `/plants/new/` | Adds a new plant. | `POST` |
| `/plants/{id}/` | Get the information about a plant identified by {id}. | `HEAD`, `GET` |

### Endpoints
#### Get the list of plants
To get the list of plants, send an HTTP `GET` request to the `/plants/` URL.

If the request method is neither `HEAD` or `GET`, the server replies with a
`405 Method Not Allowed` response. Otherwise it returns a list of plant's
identifier and common name, separated by a comma, one per line.

##### Example:

If the database contains the following plants:

```
+----+-------------+
| id | common_name |
+----+-------------+
|  1 | rosemary    |
|  2 | salvia      |
|  3 | ipomea      |
+----+-------------+
```

Then a succesful `GET /plants/` request would get the following response body:

```
1,rosemary
2,salvia
3,ipomea
```

Test with `cURL` and a Hortus API server running on your local machine:

`
curl http://localhost:8080/plants/
`

#### Add a plant
To add a plant, send an HTTP `POST` request to the `/plants/new/` URL. Currently
the API accepts three fields:

- `common-name`: The common name of the plant, must be not empty and inferior or
equal to 255 in length and UTF-8 encoded. This is a mandatory field.

- `generic-name`: The generic name of the plant, can be omitted, of length
inferior or equal to 255 and UTF-8 encoded.

- `specific-name`: The specific name of the plant, can be omitted, of length
inferior or equal to 255 and UTF-8 encoded.

If the request method is not `POST`, the server replies with the
`405 Method Not Allowed` error response. If an error occurs while parsing the
request form, the server replies with the `400 Bad Request` error response. If
an error occurs while inserting the plant, the response body contains an error
message and the status code is set to `500 Internal Server Error`.

If everything goes well, the response body contains the identifier of the newly
inserted plant in its textual form and the status code is `200 OK`.

##### Example:

Test with `cURL` and a Hortus API server running on your local machine:

`
curl -d 'common-name=passiflora' http://localhost:8080/plants/new/
`

#### Get information about a plant
To get information about a plant, send an HTTP `GET` request to the
`/plants/{id}` URL, where `{id}` is a plant identifier.

If the request method is neither `HEAD` or `GET`, the server replies with a
`405 Method Not Allowed` response. If the provided identifier is not a number,
the server replies with the `400 Bad Request` error response. If an error occurs
while fetching the plant information, the server response body contains the
error message and the status code is set to `500 Internal Server Error`.

If everything goes well, the response body contains a JSON object with relevant
plant data.

##### Example:

Suppose the identifier 42 corresponds to an Aluminium plant, *Pilea cadierei*,
stored in the database.

A succesful call to `curl http://localhost:8080/plants/42/` would return the
following JSON encoded data:

```json
{
    "id": 42,
    "commonName": "Aluminium plant",
    "genericName": "Pilea",
    "specificName": "cadierei"
}
```
