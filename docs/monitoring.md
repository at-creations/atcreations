# Error monitoring (errex)

[errex](https://github.com/TheHoltz/errex) collects errors from the API, storefront and admin. It
speaks Sentry's protocol, so the apps use the official Sentry SDKs pointed at errex. When an error is
**new**, or a resolved one **comes back**, errex calls a webhook; `alert-relay` turns that into an
email through SendGrid.

```
API / storefront / admin ─(Sentry SDK)─▶ errex ─(webhook, private network)─▶ alert-relay ─▶ SendGrid ─▶ email
```

The stack is `deploy/monitoring/docker-compose.yml`, deployed by Coolify on sr1 (the Coolify host,
not the app server, so errors are still reported if the app server is down).

## Setup

1. **DNS** (zone `baoopn.com`): `A errex → 162.250.191.124`, **DNS only**. Proxied through
   Cloudflare, Bot Fight Mode would challenge the API's server-side reports.
2. **Coolify:** AT Creations → production → + New → Public Repository
   `https://github.com/at-creations/atcreations`, branch `main`, server **localhost** (sr1),
   Build Pack **Docker Compose**, location `/deploy/monitoring/docker-compose.yml`. Auto Deploy off.
   - Domain for **errex**: `https://errex.baoopn.com:9090`; none for alert-relay.
   - Environment: `ERREX_DOMAIN=errex.baoopn.com`, `SENDGRID_API_KEY`, `MAIL_FROM` (a verified
     SendGrid sender, e.g. `no-reply@atcreations.ca`), `MAIL_TO` (comma separated).
   - Deploy. Coolify generates `SERVICE_PASSWORD_64_ERREXADMIN` (setup token) and
     `SERVICE_PASSWORD_64_RELAY` (the relay's secret path).
3. **Admin account:** open `https://errex.baoopn.com/setup`, paste the setup token, create your user.
4. **Projects and alerts** (in the errex container, Coolify → errex → Terminal):

   ```bash
   errexd project add api            # prints the DSN for that app
   errexd project add storefront
   errexd project add admin
   for p in api storefront admin; do
     errexd project set-webhook "$p" "http://alert-relay:8080/hook/<SERVICE_PASSWORD_64_RELAY>"
   done
   ```

   errex refuses webhook URLs on private IPs or `.local`/`.internal` names; the bare service name
   `alert-relay` is allowed and resolves only inside this stack's network.

5. **Apps:** set each app's DSN (see the Sentry SDK wiring in the apps).

## Operating it

- **Mute** a noisy issue in errex to stop its emails; **resolve** it and it alerts again only if
  it comes back.
- Delivery status of the last webhook shows on each project's page in errex. The relay logs
  `sent: <subject>` or `sending failed: …` (Coolify → alert-relay → Logs).
- Events are kept 30 days (`ERREX_RETENTION_DAYS`); ingest is capped at 600 events/minute per
  project.
- Relay tests: `python -m unittest deploy/monitoring/alert-relay/test_relay.py` (CI runs them).
