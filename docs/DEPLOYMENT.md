# Deploying Woodpecker Online

Options to host the Go server and SQLite-backed app on the web. The app listens on `PORT` (default `8081`) and uses a single SQLite file (`woodpecker.db`).

---

## 1. **Fly.io** (recommended for Go + SQLite)

**Best fit:** Single-region app with persistent SQLite, free tier, global edge.

- **Pros:** Native Go support, SQLite-friendly (persistent volumes), automatic HTTPS, `flyctl` CLI, good free allowance.
- **Cons:** Volume is per-machine (one region unless you use LiteFS).
- **Free tier:** Small VMs, limited monthly usage; see [Fly.io pricing](https://fly.io/docs/about/pricing/).

### Steps

1. Install [flyctl](https://fly.io/docs/flyctl/install/) and log in: `fly auth login`.
2. From the **project root** (where `go.mod` is), run:
   ```bash
   fly launch --no-deploy
   ```
   - When prompted, do **not** add Postgres/Redis.
   - Pick a region (e.g. `iad` for US East).
3. Add a **volume** for SQLite (after first deploy):
   ```bash
   fly volumes create woodpecker_data --size 1
   ```
4. In `fly.toml`, set the port and add a mount (create `fly.toml` in project root if needed):

   ```toml
   app = "woodpecker-online"

   [env]
     PORT = "8080"

   [http_service]
     internal_port = 8080
     force_https = true
     auto_stop_machines = "stop"
     auto_start_machines = true
     min_machines_running = 0
     processes = ["app"]

   [mounts]
     source = "woodpecker_data"
     destination = "/data"

   [[vm]]
     memory = "512mb"
     cpu_kind = "shared"
     cpus = 1
   ```

5. Point the app at the volume: set env or code so the SQLite path is `/data/woodpecker.db` (see “App changes” below).
6. Deploy:
   ```bash
   fly deploy
   fly open
   ```

**SQLite path:** Configure the app to use `DATABASE_PATH` or similar (e.g. `/data/woodpecker.db` on Fly, `./woodpecker.db` locally). The server already supports this if you add the env check.

**Backups:** Use [LiteStream](https://fly.io/docs/litefs/) or periodic `fly ssh console` + `sqlite3 /data/woodpecker.db ".backup stdout"` to a file.

---

## 2. **Railway**

**Best fit:** Git-based deploys, minimal config.

- **Pros:** Nixpacks detects Go, GitHub auto-deploy, simple dashboard.
- **Cons:** Free trial then ~$5/month or pay-as-you-go; SQLite on ephemeral disk loses data on redeploy unless you add a **volume** (Railway offers persistent volumes).
- **Docs:** [Railway – Deploy from GitHub](https://docs.railway.app/deploy/deploy-from-github), [Volumes](https://docs.railway.app/reference/volumes).

### Steps

1. Push the repo to GitHub.
2. In [Railway](https://railway.app), New Project → Deploy from GitHub repo.
3. Add a **Volume** and mount it (e.g. `/data`). Set env (or code) so SQLite path is `/data/woodpecker.db`.
4. Set `PORT` (Railway sets this automatically; the app reads it).
5. Redeploy; DB persists on the volume.

---

## 3. **Render**

**Best fit:** Free hobby tier, good for demos.

- **Pros:** Free tier, managed SSL, native Go.
- **Cons:** Free web services spin down after inactivity (cold starts); free **disk** is ephemeral—SQLite will be lost on redeploy unless you use a [persistent disk](https://render.com/docs/disks) (paid).
- **Docs:** [Render – Go](https://render.com/docs/deploy-go), [Free tier](https://render.com/docs/free).

### Steps

1. Connect the GitHub repo in Render.
2. New → Web Service, select the repo.
3. **Build:** `go build -o server ./cmd/server` (or set root to `cmd/server` and build from there).
4. **Start:** `./server` (Render sets `PORT`).
5. For persistent SQLite, add a **Disk** (paid), mount it, and use that path for `woodpecker.db`.

---

## 4. **VPS (DigitalOcean, Linode, Vultr, etc.)**

**Best fit:** Full control, always-on, predictable cost.

- **Pros:** Full control, SQLite works normally on local disk, no “spin down”, simple backups (copy DB file).
- **Cons:** You manage OS, HTTPS (e.g. Caddy/Nginx), and updates.
- **Cost:** ~$4–6/month for a small droplet.

### Steps

1. Create a small Ubuntu VPS (e.g. 1 GB RAM).
2. Install Go (or build on CI and copy the binary), clone repo, build:
   ```bash
   go build -o woodpecker ./cmd/server
   ```
3. Run with `PORT=80` (or use a reverse proxy and keep `PORT=8080`).
4. Use **systemd** (or supervisor) to keep the process running.
5. Put Caddy or Nginx in front for HTTPS (e.g. Caddy with Let’s Encrypt).
6. Back up `woodpecker.db` regularly (cron + `scp` or object storage).

---

## 5. **Other options (short list)**

| Platform      | Go support | SQLite / persistence        | Note                    |
|---------------|------------|-----------------------------|--------------------------|
| **Koyeb**     | Yes        | Ephemeral unless you add storage | Serverless-style         |
| **Google App Engine** | Yes | Not standard for SQLite    | Prefer Cloud SQL or adapt |
| **AWS / Azure / GCP** | Yes  | Via VM + disk or RDS        | More setup, flexible     |
| **Coolify / CapRover** | Yes | Your server, your disk   | Self-hosted PaaS         |

---

## App configuration (already supported)

1. **Port:** Set `PORT` (e.g. `8080`). Fly/Railway/Render set this automatically. Default: `8081`.
2. **Database path:** Set `DATABASE_PATH` to the SQLite file path:
   - Local: leave unset → uses `woodpecker.db`.
   - Production (with volume/disk): e.g. `DATABASE_PATH=/data/woodpecker.db`.

---

## Quick comparison

| Criteria           | Fly.io     | Railway   | Render (free) | VPS        |
|-------------------|------------|-----------|----------------|------------|
| Go + SQLite       | ✅ Volumes | ✅ Volumes | ⚠️ Paid disk   | ✅ Native  |
| Free / cheap tier | ✅         | Trial then $ | ✅ (spin down) | ❌ ~$4/mo  |
| HTTPS             | ✅         | ✅        | ✅             | You config |
| Ease of deploy    | CLI + toml | Git push  | Git push       | Manual     |

**Recommendation:** Use **Fly.io** for a simple, always-available deployment with persistent SQLite and HTTPS; use a **VPS** if you prefer full control and don’t mind configuring the server and TLS yourself.
