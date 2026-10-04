# Leave Report Generator

Fill in a leave application, keep track of the yearly balance, and print it as a PDF
in the company's leave-form layout.

- **Backend:** Go (`net/http`) + SQLite (`modernc.org/sqlite`, pure Go)
- **Frontend:** React + Vite
- **Deployment:** Docker (a single container; Go serves the API and the built React app)

## Run with Docker (recommended)

```bash
docker compose up -d --build
```

Open http://localhost:8080. The SQLite database is stored in the `leave-data` Docker volume,
so your data survives restarts and rebuilds.

```bash
docker compose logs -f     # view logs
docker compose down        # stop (data is kept)
docker compose down -v     # stop AND delete all data
```

## Deploy on a server

See **[DEPLOY.md](DEPLOY.md)** for step-by-step instructions (Docker install, HTTPS with a password, updates, backups).

## Run locally for development

Requires Go 1.26+ and Node 24+. Use two terminals:

```bash
# Terminal 1 – API on :8080 (creates backend/leave.db)
cd backend
go run .

# Terminal 2 – React dev server on :5173 (proxies /api to :8080)
cd frontend
npm install
npm run dev
```

Open http://localhost:5173.

## How the balance works

| Term | Meaning |
|---|---|
| Eligibility | Always 21 days per calendar year |
| Already used (opening) | Days taken before you started using this app. Set under **Already used leave** for each year |
| Used | Opening days + earlier non-Saturday leaves in the same year |
| This time | Number of days of this application, or **0 if it is a Saturday leave** |
| Remaining | Eligibility − Used − This time |

The year is taken from the **Holiday starts from** date.

## Employee name and number

Saved in the database. Change them at any time and click **Save employee**; they are also saved
automatically whenever you submit a leave. Earlier applications keep the name they were printed with.

## Making the PDF

**Save & Download PDF** saves the application and opens the print dialog. Choose
**Save as PDF**, and turn off *Headers and footers* under "More settings".
Use the **PDF** button in the history list to print an earlier application again.

## Configuration (environment variables)

| Variable | Default | Purpose |
|---|---|---|
| `DB_PATH` | `leave.db` (`/data/leave.db` in Docker) | SQLite file location |
| `ADDR` | `:8080` | Listen address |
| `STATIC_DIR` | empty (`/app/web` in Docker) | Folder with the built React app |
| `TZ` | `Asia/Dhaka` in Docker | Time zone used for the letter's date |

## Backup

```bash
docker compose stop
docker compose run --rm --no-deps --user root -v "$PWD":/backup leave-report \
  cp /data/leave.db /backup/leave-backup.db
docker compose start
```

Restoring is described in [DEPLOY.md](DEPLOY.md#backups).
