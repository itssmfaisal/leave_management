# Deploying to a server

These steps assume a Linux server (Ubuntu 22.04 / 24.04) that you can reach over SSH.
The app is small: 1 CPU and 512 MB RAM is plenty.

> **Important:** the app has no login. Anyone who can open the URL can view and change the data.
> Either keep it on your office network/VPN, or follow step 6 to add a password and HTTPS.

---

## 1. Install Docker on the server

```bash
ssh user@your-server
curl -fsSL https://get.docker.com | sh
sudo usermod -aG docker $USER
exit            # log out and back in so the group change applies
```

Check it works:

```bash
ssh user@your-server
docker compose version
```

## 2. Copy the project to the server

From your Mac, in the project folder:

```bash
rsync -av --exclude node_modules --exclude dist --exclude '*.db*' \
  ~/Documents/leave_report_gen/ user@your-server:~/leave-report/
```

(Alternatively, push the project to a private Git repository and `git clone` it on the server.)

## 3. Start the app

```bash
ssh user@your-server
cd ~/leave-report
docker compose up -d --build
docker compose ps        # STATUS should be "Up"
```

The app restarts automatically after a reboot or crash (`restart: unless-stopped`).

## 4. Open it

If you are **not** using a domain/HTTPS (office network only), allow the port through the firewall:

```bash
sudo ufw allow 8080/tcp
```

Then browse to `http://SERVER_IP:8080`.

## 5. First-time setup in the app

1. Enter your **Name** and **Employee Number** and click **Save employee**.
2. Under **Already used leave**, enter the days you have already taken this year and click **Save**.

---

## 6. (Recommended) Domain, HTTPS and a password with Caddy

You need a domain name (e.g. `leave.example.com`) whose DNS **A record** points to the server's IP.

**a. Make the app reachable only from the server itself.** In `~/leave-report`, create a file named `.env`:

```bash
echo "APP_PORT=127.0.0.1:8080" > ~/leave-report/.env
cd ~/leave-report && docker compose up -d
```

**b. Install Caddy:**

```bash
sudo apt install -y debian-keyring debian-archive-keyring apt-transport-https curl
curl -1sLf 'https://dl.cloudsmith.io/public/caddy/stable/gpg.key' | sudo gpg --dearmor -o /usr/share/keyrings/caddy-stable-archive-keyring.gpg
curl -1sLf 'https://dl.cloudsmith.io/public/caddy/stable/debian.deb.txt' | sudo tee /etc/apt/sources.list.d/caddy-stable.list
sudo apt update && sudo apt install -y caddy
```

**c. Create a password hash:**

```bash
caddy hash-password        # type your password; copy the output starting with $2a$...
```

**d. Edit `/etc/caddy/Caddyfile`** (`sudo nano /etc/caddy/Caddyfile`), replacing everything with:

```
leave.example.com {
    basic_auth {
        admin PASTE_THE_HASH_HERE
    }
    reverse_proxy 127.0.0.1:8080
}
```

**e. Open the web ports and reload Caddy:**

```bash
sudo ufw allow 80/tcp
sudo ufw allow 443/tcp
sudo ufw delete allow 8080/tcp   # only if you opened it in step 4
sudo systemctl reload caddy
```

Browse to `https://leave.example.com`. Caddy gets and renews the HTTPS certificate automatically.
Log in with user `admin` and your password.

---

## Updating to a new version

From your Mac, copy the changed code again, then rebuild on the server:

```bash
rsync -av --exclude node_modules --exclude dist --exclude '*.db*' --exclude .env \
  ~/Documents/leave_report_gen/ user@your-server:~/leave-report/

ssh user@your-server "cd ~/leave-report && docker compose up -d --build"
```

Your data is kept: it lives in the `leave-data` Docker volume, not in the project folder.

## Backups

Make a backup on the server (the app is stopped for a few seconds):

```bash
cd ~/leave-report
docker compose stop
docker compose run --rm --no-deps --user root -v "$PWD":/backup leave-report \
  cp /data/leave.db /backup/leave-backup.db
docker compose start
```

Download it to your Mac:

```bash
scp user@your-server:~/leave-report/leave-backup.db ~/Desktop/
```

To restore, put `leave-backup.db` in `~/leave-report` on the server, then:

```bash
cd ~/leave-report
docker compose stop
docker compose run --rm --no-deps --user root -v "$PWD":/backup leave-report sh -c \
  'cp /backup/leave-backup.db /data/leave.db && rm -f /data/leave.db-wal /data/leave.db-shm && chown app /data/leave.db'
docker compose start
```

## Useful commands

| Task | Command (run in `~/leave-report`) |
|---|---|
| See if it's running | `docker compose ps` |
| View logs | `docker compose logs -f` |
| Restart | `docker compose restart` |
| Stop | `docker compose down` (data is kept) |
| **Delete everything including data** | `docker compose down -v` |

## Troubleshooting

- **Page doesn't load:** check `docker compose ps` and `docker compose logs`, and that the firewall
  allows the port (8080, or 80/443 with Caddy).
- **Wrong date on the letter:** the date uses `TZ` in `docker-compose.yml` (default `Asia/Dhaka`).
- **HTTPS certificate fails:** make sure the domain's DNS points to the server and ports 80/443 are open,
  then check `sudo journalctl -u caddy -n 50`.
