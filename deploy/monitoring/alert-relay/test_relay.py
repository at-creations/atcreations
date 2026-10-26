"""Tests for the errex -> email relay: python -m unittest discover deploy/monitoring/alert-relay"""

import json
import os
import threading
import unittest
import urllib.error
import urllib.request
from http.server import ThreadingHTTPServer
from unittest import mock

import relay

PAYLOAD = {
    "text": "New issue: TypeError: x is undefined",
    "attachments": [
        {
            "title": "TypeError: x is undefined",
            "title_link": "https://errex.example.com/issues/42",
            "color": "danger",
            "fields": [
                {"title": "Project", "value": "storefront", "short": True},
                {"title": "Events", "value": "1", "short": True},
                {"title": "Culprit", "value": "<script>alert(1)</script>", "short": False},
            ],
            "fallback": "New issue: TypeError: x is undefined",
            "ts": 1792990000,
        }
    ],
}


class BuildEmail(unittest.TestCase):
    def test_subject_body_and_link(self):
        subject, text, body = relay.build_email(PAYLOAD)
        self.assertEqual(subject, "New issue: TypeError: x is undefined")
        self.assertIn("Project: storefront", text)
        self.assertIn("Open in errex: https://errex.example.com/issues/42", text)
        self.assertIn('href="https://errex.example.com/issues/42"', body)

    def test_html_is_escaped(self):
        _, _, body = relay.build_email(PAYLOAD)
        self.assertNotIn("<script>", body)
        self.assertIn("&lt;script&gt;", body)

    def test_minimal_payload(self):
        subject, _, _ = relay.build_email({})
        self.assertEqual(subject, "errex alert")


class Providers(unittest.TestCase):
    def capture(self, env):
        sent = {}

        class Resp:
            status = 200

            def __enter__(self):
                return self

            def __exit__(self, *a):
                return False

        def fake_urlopen(req, timeout):
            sent["url"], sent["auth"] = req.full_url, req.get_header("Authorization")
            sent["ua"] = req.get_header("User-agent")
            sent["body"] = json.loads(req.data)
            return Resp()

        with mock.patch.dict(os.environ, env, clear=True), mock.patch.object(
            relay.urllib.request, "urlopen", fake_urlopen
        ):
            relay.send_mail("Subject", "text", "<p>html</p>")
        return sent

    def test_usesend_is_the_default(self):
        sent = self.capture({"USESEND_URL": "https://us.example.com/", "USESEND_API_KEY": "us_k",
                             "MAIL_FROM": "a@b.c", "MAIL_TO": "d@e.f, g@h.i"})
        self.assertEqual(sent["url"], "https://us.example.com/api/v1/emails")
        self.assertEqual(sent["auth"], "Bearer us_k")
        self.assertEqual(sent["ua"], relay.USER_AGENT)
        self.assertEqual(sent["body"]["to"], ["d@e.f", "g@h.i"])
        self.assertEqual(sent["body"]["from"], "errex alerts <a@b.c>")
        self.assertEqual((sent["body"]["text"], sent["body"]["html"]), ("text", "<p>html</p>"))

    def test_sendgrid(self):
        sent = self.capture({"MAIL_PROVIDER": "sendgrid", "SENDGRID_API_KEY": "SG.k",
                             "MAIL_FROM": "a@b.c", "MAIL_TO": "d@e.f"})
        self.assertEqual(sent["url"], relay.SENDGRID_URL)
        self.assertEqual(sent["body"]["personalizations"][0]["to"], [{"email": "d@e.f"}])


class Server(unittest.TestCase):
    TOKEN = "t" * 32

    def setUp(self):
        env = {"RELAY_TOKEN": self.TOKEN, "SENDGRID_API_KEY": "k", "MAIL_FROM": "a@b.c", "MAIL_TO": "d@e.f"}
        self.env = mock.patch.dict(os.environ, env)
        self.env.start()
        self.srv = ThreadingHTTPServer(("127.0.0.1", 0), relay.Handler)
        threading.Thread(target=self.srv.serve_forever, daemon=True).start()
        self.base = f"http://127.0.0.1:{self.srv.server_address[1]}"

    def tearDown(self):
        self.srv.shutdown()
        self.env.stop()

    def post(self, path, data):
        req = urllib.request.Request(self.base + path, data=data, method="POST")
        try:
            return urllib.request.urlopen(req, timeout=5).status
        except urllib.error.HTTPError as err:
            return err.code

    def test_wrong_token_is_404_and_sends_nothing(self):
        with mock.patch.object(relay, "send_mail") as send:
            self.assertEqual(self.post("/hook/wrong", json.dumps(PAYLOAD).encode()), 404)
            send.assert_not_called()

    def test_valid_hook_sends_one_mail(self):
        with mock.patch.object(relay, "send_mail") as send:
            self.assertEqual(self.post(f"/hook/{self.TOKEN}", json.dumps(PAYLOAD).encode()), 204)
            send.assert_called_once()

    def test_bad_json_is_400(self):
        self.assertEqual(self.post(f"/hook/{self.TOKEN}", b"{nope"), 400)

    def test_send_failure_is_502(self):
        with mock.patch.object(relay, "send_mail", side_effect=RuntimeError("boom")):
            self.assertEqual(self.post(f"/hook/{self.TOKEN}", json.dumps(PAYLOAD).encode()), 502)

    def test_healthz(self):
        self.assertEqual(urllib.request.urlopen(self.base + "/healthz", timeout=5).status, 200)


if __name__ == "__main__":
    unittest.main()
