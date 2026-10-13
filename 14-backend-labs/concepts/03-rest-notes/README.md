# Lesson 03: notes with PostgreSQL

Create, list, update, and delete are implemented. Pagination and authentication come later.

## Current setup: Neon

The database is hosted on Neon. Use the Neon connection URI in DATABASE_URL; no Supabase roles or services are required. The table requires no changes for update/delete.

Restart the Go server in the terminal that has DATABASE_URL set, then refresh http://127.0.0.1:8080/concepts/notes. Each note has Edit and Delete buttons. Edit opens an inline field with Save changes/Cancel; Delete shows inline confirmation. These send PUT and DELETE respectively. The last API response remains visible after refreshing the list.

| Request | Success | Missing note |
| --- | --- | --- |
| PUT /api/notes/{id} | 200 with updated note | 404 |
| DELETE /api/notes/{id} | 204 with no body | 404 |

Invalid IDs and blank text return 400. SQL uses placeholders, never concatenated user input. readText shares bounded JSON validation; readID extracts and validates the path ID; writeJSON shares response formatting. Validation retains the lesson 2 bounded read plus Unmarshal approach rather than introducing decoder/EOF checks. Existing explanatory comments are retained.

Browser → local Go API → Neon-hosted PostgreSQL.

## Setup, with reasons

1. In Neon, create a separate project named `go-backend-mastery`, enabling only PostgreSQL. Choose a nearby region. Do not reuse databases belonging to other apps.
2. Open the project's **SQL Editor**, select `neondb`, paste `schema.sql` from this folder, and run it. This creates a namespace, a notes table, generated IDs, and a constraint rejecting empty text. This local learning setup uses the database owner; a restricted backend role is a production follow-up.
3. Open **Connect** and copy the PostgreSQL URI for the same database and branch. Keep its TLS settings. This is a database URI, not a website URL. Do not send it in chat.
4. Put the URI in a local environment variable. In zsh, this prompts without displaying the password or storing it as a literal command in shell history:

   ```sh
   read -rs 'DATABASE_URL?Paste database URI (hidden): '; echo
   export DATABASE_URL
   ```

   Keep TLS enabled and retain the connection settings Neon provides. Do not disable security to fix connection errors.
5. From the repository root run:

   ```sh
   go run ./14-backend-labs/cmd/server
   ```

   Startup opens a connection pool, limits it to five connections, and pings PostgreSQL with a ten-second deadline. The URI is never printed. Without DATABASE_URL, lessons 1/2 work but lesson 3's API is disabled. The code does not automatically load `.env` files.
6. Open **http://127.0.0.1:8080/concepts/notes**. Type a short note and click Save note. The page sends POST JSON to Go; Go validates it, executes INSERT, and returns the saved ID and text with status 201. Refresh notes executes SELECT through Go, returning status 200. The page shows the real API response, not simulated rows.
7. Stop Go with Control+C, run it again in the same terminal, and refresh the page. Notes should remain because PostgreSQL owns the durable rows. A new terminal needs the environment variable set again. In Neon SQL Editor, `SELECT id, text FROM backend_labs.notes ORDER BY id;` shows the same rows independently of Go.

No credentials are sent to the browser. `.env` files are ignored as a precaution, but do not commit secrets or paste them into screenshots. Keep the Go server bound to localhost. This demo has no authentication and must not be publicly deployed as-is.

## Reading the code

`Note` contains ID and Text. JSON tags control their outgoing field names.

`Store` now contains `db *sql.DB`, a pointer to a database connection pool. It no longer stores a slice or nextID. SQL connections can be shared by concurrent requests; PostgreSQL handles ID allocation. `NewStore(db *sql.DB) *Store` receives that pool and returns a store pointing to it.

`func (s *Store) Create(w http.ResponseWriter, r *http.Request)` is a method. `s` is its receiver, the store it operates on. `w` writes the response; `r` supplies the request. The method returns nothing.

Create reuses lesson 2's content-type check, bounded body read, JSON decoding, and blank-text validation. Unknown JSON fields are ignored. It then runs:

```sql
INSERT INTO backend_labs.notes (text) VALUES ($1) RETURNING id, text
```

`$1` is a placeholder whose value is supplied separately as input.Text. User text is data, not executable SQL. PostgreSQL generates the identity ID. RETURNING supplies the saved row. `QueryRowContext(...).Scan(&note.ID, &note.Text)` copies its two columns into the fields; Scan needs addresses because it modifies them. A single successful INSERT is committed automatically, so it survives the Go process ending. IDs can have gaps; they are not row counts.

List runs SELECT ordered by ID, closes rows when finished, calls Next for each row, and Scan to fill a Note. The temporary slice only collects query results for the JSON response; it is not the persistent storage. rows.Err checks for failures during iteration. Query errors return generic 500 responses without leaking database details.

`context.WithTimeout(r.Context(), 5*time.Second)` bounds waiting and inherits request cancellation. `defer cancel()` releases timer resources. Context details will get a focused lesson later.

`sql.Open("pgx", databaseURL)` configures a pool; PingContext actually verifies connectivity. The blank pgx import registers its PostgreSQL driver. Database operations go through Go's database/sql package, not Supabase's generated REST API.

Register attaches GET/POST collection routes and PUT/DELETE routes with an ID to ServeMux. Unsupported methods receive router-generated 405 responses. GET also matches HEAD. The schema SQL is run explicitly, not recreated or wiped on each startup.

## Tests and limits

```sh
go test -v ./14-backend-labs/concepts/03-rest-notes
```

Tests use sqlmock to simulate query responses, not an in-memory production store. They check validation, SQL arguments, returned rows, and database-failure handling without credentials. They do not prove SQL syntax, Neon connectivity, or durability; the real setup/restart exercise above verifies those separately.

The list endpoint currently returns all notes, suitable only for a small learning dataset. Pagination, least-privilege database roles, migrations, and backup/recovery are follow-ups. Restarting Go does not delete rows; deleting the project/table or its data still does. Persistence is not a replacement for backups.

Connection reference: [Neon documentation](https://neon.com/docs/get-started/connect-neon).
