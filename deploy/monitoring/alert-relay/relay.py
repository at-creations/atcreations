"""errex webhook -> email relay.

errex (https://github.com/TheHoltz/errex) only alerts through Slack-style webhooks. This
service receives that webhook on a secret path and sends one email per alert through
SendGrid. It is reachable only on the monitoring stack's private Docker network.

Environment:
  RELAY_TOKEN       secret path segment: errex posts to http://alert-relay:8080/hook/<token>
  SENDGRID_API_KEY  SendGrid key allowed to send mail
  MAIL_FROM         verified sender, e.g. no-reply@atcreations.ca
  MAIL_TO           recipient(s), comma separated
  PORT              listen port (default 8080)
"""

import hmac
import html
import json
import logging
import os
import sys
import urllib.error
import urllib.request
from datetime import datetime, timezone
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer

MAX_BODY = 64 * 1024
SENDGRID_URL = "https://api.sendgrid.com/v3/mail/send"

log = logging.getLogger("alert-relay")


def build_email(payload: dict) -> tuple[str, str, str]:
    """Subject, plain-text and HTML bodies from errex's Slack-compatible payload."""
    att = (payload.get("attachments") or [{}])[0]
    subject = str(payload.get("text") or att.get("fallback") or "errex alert")[:200]
    title = str(att.get("title") or subject)
    link = str(att.get("title_link") or "")
    fields = [
        (str(f.get("title", "")), str(f.get("value", "")))
        for f in att.get("fields") or []
        if isinstance(f, dict)
    ]
    ts = att.get("ts")
    when = (
        datetime.fromtimestamp(ts, timezone.utc).strftime("%Y-%m-%d %H:%M UTC")
        if isinstance(ts, (int, float))
        else ""
    )

    lines = [subject, "", title, ""]
    lines += [f"{k}: {v}" for k, v in fields]
    if when:
        lines.append(f"Last seen: {when}")
    if link:
        lines += ["", f"Open in errex: {link}"]
    text = "\n".join(lines)

    rows = "".join(
        f'<tr><td style="padding:2px 12px 2px 0;color:#71717a">{html.escape(k)}</td>'
        f"<td>{html.escape(v)}</td></tr>"
        for k, v in fields + ([("Last seen", when)] if when else [])
    )
    button = (
        f'<p><a href="{html.escape(link, quote=True)}" style="color:#5a4fb0">Open in errex</a></p>'
        if link
        else ""
    )
    body = (
        '<div style="font-family:system-ui,sans-serif;font-size:14px;color:#18181b">'
        f'<p style="font-size:16px;font-weight:600;margin:0 0 8px">{html.escape(title)}</p>'
        f'<table style="border-collapse:collapse;margin:0 0 12px">{rows}</table>{button}</div>'
    )
    return subject, text, body


def send_mail(subject: str, text: str, body: str) -> None:
    recipients = [a.strip() for a in os.environ["MAIL_TO"].split(",") if a.strip()]
    message = {
        "personalizations": [{"to": [{"email": a} for a in recipients]}],
        "from": {"email": os.environ["MAIL_FROM"], "name": "errex alerts"},
        "subject": subject,
        "content": [
            {"type": "text/plain", "value": text},
            {"type": "text/html", "value": body},
        ],
        # Alerts are operational mail: no click or open tracking rewriting the links.
        "tracking_settings": {
            "click_tracking": {"enable": False, "enable_text": False},
            "open_tracking": {"enable": False},
        },
    }
    req = urllib.request.Request(
        SENDGRID_URL,
        data=json.dumps(message).encode(),
        method="POST",
        headers={
            "Authorization": f"Bearer {os.environ['SENDGRID_API_KEY']}",
            "Content-Type": "application/json",
        },
    )
    with urllib.request.urlopen(req, timeout=15) as resp:  # noqa: S310 (fixed https URL)
        if resp.status >= 300:
            raise RuntimeError(f"SendGrid returned {resp.status}")


class Handler(BaseHTTPRequestHandler):
    server_version = "alert-relay"
    sys_version = ""

    def _reply(self, code: int) -> None:
        self.send_response(code)
        self.send_header("Content-Length", "0")
        self.end_headers()

    def do_GET(self) -> None:
        self._reply(200 if self.path == "/healthz" else 404)

    def do_POST(self) -> None:
        try:
            length = int(self.headers.get("Content-Length") or 0)
        except ValueError:
            length = -1
        if not 0 < length <= MAX_BODY:
            self.close_connection = True
            self._reply(413 if length > MAX_BODY else 400)
            return
        # Read the (bounded) body before answering, so a rejected client gets a clean reply
        # instead of a connection reset.
        raw = self.rfile.read(length)
        token = os.environ["RELAY_TOKEN"]
        prefix = "/hook/"
        given = self.path[len(prefix) :] if self.path.startswith(prefix) else ""
        if not given or not hmac.compare_digest(given.encode(), token.encode()):
            self._reply(404)
            return
        try:
            payload = json.loads(raw)
            if not isinstance(payload, dict):
                raise ValueError("payload is not an object")
        except ValueError:
            self._reply(400)
            return
        subject, text, body = build_email(payload)
        try:
            send_mail(subject, text, body)
        except (urllib.error.URLError, RuntimeError, TimeoutError) as err:
            log.error("sending failed: %s", err)
            self._reply(502)
            return
        log.info("sent: %s", subject)
        self._reply(204)

    def log_message(self, fmt: str, *args) -> None:  # keep the secret path out of the logs
        pass


def main() -> None:
    logging.basicConfig(level=logging.INFO, format="%(asctime)s %(levelname)s %(message)s")
    for name in ("RELAY_TOKEN", "SENDGRID_API_KEY", "MAIL_FROM", "MAIL_TO"):
        if not os.environ.get(name):
            sys.exit(f"{name} is required")
    if len(os.environ["RELAY_TOKEN"]) < 24:
        sys.exit("RELAY_TOKEN must be at least 24 characters")
    port = int(os.environ.get("PORT", "8080"))
    log.info("listening on :%d", port)
    ThreadingHTTPServer(("0.0.0.0", port), Handler).serve_forever()


if __name__ == "__main__":
    main()
