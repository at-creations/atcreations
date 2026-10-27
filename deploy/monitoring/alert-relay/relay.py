"""errex webhook -> email relay.

errex (https://github.com/TheHoltz/errex) only alerts through Slack-style webhooks. This
service receives that webhook on a secret path and sends one email per alert through
useSend (self-hosted) or SendGrid. It is reachable only on the monitoring stack's private
Docker network.

Environment:
  RELAY_TOKEN       secret path segment: errex posts to http://alert-relay:8080/hook/<token>
  MAIL_PROVIDER     usesend (default) | sendgrid
  USESEND_URL       useSend base URL, e.g. https://usesend.example.com   (usesend)
  USESEND_API_KEY   useSend API key                                       (usesend)
  SENDGRID_API_KEY  SendGrid key allowed to send mail                     (sendgrid)
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
USER_AGENT = "atcreations-alert-relay/1.0"

log = logging.getLogger("alert-relay")


# errex sends "danger" for new issues and "warning" for regressions: (accent, badge background).
ACCENTS = {
    "danger": ("#c0262d", "#fdecec"),
    "warning": ("#a65a00", "#fdf1e1"),
    "good": ("#1d7a46", "#e6f4ec"),
}
NEUTRAL = ("#52525b", "#f1f1f3")
SANS = "-apple-system,BlinkMacSystemFont,'Segoe UI',Roboto,Helvetica,Arial,sans-serif"
MONO = "ui-monospace,SFMono-Regular,Menlo,Consolas,'Liberation Mono',monospace"


def build_email(payload: dict) -> tuple[str, str, str]:
    """Subject, plain-text and HTML bodies from errex's Slack-compatible payload."""
    att = (payload.get("attachments") or [{}])[0]
    headline = str(payload.get("text") or att.get("fallback") or "errex alert")
    title = str(att.get("title") or headline)
    # The headline is "<kind>: <issue title>"; the kind becomes the badge.
    kind = headline[: -len(title)].rstrip(": ") if headline.endswith(title) else ""
    link = str(att.get("title_link") or "")
    fields = {
        str(f.get("title", "")): str(f.get("value", ""))
        for f in att.get("fields") or []
        if isinstance(f, dict)
    }
    project = fields.pop("Project", "")
    culprit = fields.pop("Culprit", "")
    ts = att.get("ts")
    if isinstance(ts, (int, float)):
        fields["Last seen"] = datetime.fromtimestamp(ts, timezone.utc).strftime("%b %d, %H:%M UTC")
    subject = (f"[{project}] " if project else "") + headline
    subject = subject[:200]

    lines = [headline, ""]
    if project:
        lines.append(f"Project: {project}")
    if culprit:
        lines.append(f"Culprit: {culprit}")
    lines += [f"{k}: {v}" for k, v in fields.items()]
    if link:
        lines += ["", f"Open in errex: {link}"]
    text = "\n".join(lines)

    accent, tint = ACCENTS.get(str(att.get("color") or ""), NEUTRAL)
    e = html.escape
    badge = (
        f'<span style="display:inline-block;padding:3px 10px;border-radius:999px;background:{tint};'
        f'color:{accent};font-size:12px;font-weight:600">{e(kind)}</span>'
        if kind
        else ""
    )
    project_label = (
        f'<span style="color:#71717a;font-size:13px;padding-left:8px">{e(project)}</span>'
        if project
        else ""
    )
    culprit_block = (
        f'<p style="margin:10px 0 0;font-family:{MONO};font-size:13px;color:#52525b;'
        f'word-break:break-word">{e(culprit)}</p>'
        if culprit
        else ""
    )
    stats = "".join(
        f'<td style="padding:0 24px 0 0;vertical-align:top">'
        f'<div style="font-size:12px;color:#71717a;padding-bottom:2px">{e(k)}</div>'
        f'<div style="font-size:15px;font-weight:600;color:#18181b">{e(v)}</div></td>'
        for k, v in fields.items()
    )
    stats_block = (
        '<table role="presentation" cellpadding="0" cellspacing="0" style="margin:20px 0 0">'
        f"<tr>{stats}</tr></table>"
        if stats
        else ""
    )
    button = (
        '<table role="presentation" cellpadding="0" cellspacing="0" style="margin:24px 0 0"><tr>'
        f'<td style="border-radius:6px;background:#18181b">'
        f'<a href="{e(link, quote=True)}" style="display:inline-block;padding:10px 18px;'
        f'color:#ffffff;font-size:14px;font-weight:600;text-decoration:none">Open in errex</a>'
        "</td></tr></table>"
        if link
        else ""
    )
    preheader = e(" / ".join(x for x in (project, culprit) if x))
    body = (
        f'<div style="display:none;max-height:0;overflow:hidden">{preheader}</div>'
        '<table role="presentation" width="100%" cellpadding="0" cellspacing="0" '
        f'style="background:#f4f4f5;font-family:{SANS};color:#18181b"><tr>'
        '<td align="center" style="padding:32px 16px">'
        '<table role="presentation" width="100%" cellpadding="0" cellspacing="0" '
        'style="max-width:560px;background:#ffffff;border:1px solid #e4e4e7;border-radius:10px">'
        f'<tr><td style="height:4px;background:{accent};border-radius:10px 10px 0 0;'
        'font-size:0;line-height:0">&nbsp;</td></tr>'
        '<tr><td style="padding:24px 28px 28px">'
        f'<div>{badge}{project_label}</div>'
        f'<h1 style="margin:16px 0 0;font-family:{MONO};font-size:17px;line-height:1.45;'
        f'font-weight:600;color:#18181b;word-break:break-word">{e(title)}</h1>'
        f"{culprit_block}{stats_block}{button}"
        "</td></tr></table>"
        '<p style="margin:16px 0 0;font-size:12px;color:#a1a1aa">'
        "Sent by errex because this address is on the alert list.</p>"
        "</td></tr></table>"
    )
    return subject, text, body


def provider() -> str:
    return os.environ.get("MAIL_PROVIDER", "usesend").strip().lower()


def recipients() -> list[str]:
    return [a.strip() for a in os.environ["MAIL_TO"].split(",") if a.strip()]


def post_json(url: str, token: str, message: dict, service: str) -> None:
    req = urllib.request.Request(
        url,
        data=json.dumps(message).encode(),
        method="POST",
        headers={
            "Authorization": f"Bearer {token}",
            "Content-Type": "application/json",
            # Cloudflare's Browser Integrity Check blocks urllib's default User-Agent (error 1010)
            # in front of a proxied useSend.
            "User-Agent": USER_AGENT,
        },
    )
    with urllib.request.urlopen(req, timeout=15) as resp:  # noqa: S310 (configured https URL)
        if resp.status >= 300:
            raise RuntimeError(f"{service} returned {resp.status}")


def send_mail(subject: str, text: str, body: str) -> None:
    if provider() == "sendgrid":
        send_sendgrid(subject, text, body)
    else:
        send_usesend(subject, text, body)


def send_usesend(subject: str, text: str, body: str) -> None:
    message = {
        "to": recipients(),
        "from": f"errex alerts <{os.environ['MAIL_FROM']}>",
        "subject": subject,
        "text": text,
        "html": body,
    }
    url = os.environ["USESEND_URL"].rstrip("/") + "/api/v1/emails"
    post_json(url, os.environ["USESEND_API_KEY"], message, "useSend")


def send_sendgrid(subject: str, text: str, body: str) -> None:
    message = {
        "personalizations": [{"to": [{"email": a} for a in recipients()]}],
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
    post_json(SENDGRID_URL, os.environ["SENDGRID_API_KEY"], message, "SendGrid")


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
    required = {
        "usesend": ("USESEND_URL", "USESEND_API_KEY"),
        "sendgrid": ("SENDGRID_API_KEY",),
    }
    if provider() not in required:
        sys.exit("MAIL_PROVIDER must be usesend or sendgrid")
    for name in ("RELAY_TOKEN", "MAIL_FROM", "MAIL_TO", *required[provider()]):
        if not os.environ.get(name):
            sys.exit(f"{name} is required")
    if len(os.environ["RELAY_TOKEN"]) < 24:
        sys.exit("RELAY_TOKEN must be at least 24 characters")
    port = int(os.environ.get("PORT", "8080"))
    log.info("listening on :%d", port)
    ThreadingHTTPServer(("0.0.0.0", port), Handler).serve_forever()


if __name__ == "__main__":
    main()
