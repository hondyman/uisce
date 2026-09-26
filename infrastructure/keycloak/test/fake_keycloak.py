"""A tiny stand-in for the Keycloak endpoints rotate-admin-password.sh uses.

Usage: fake_keycloak.py PORT PASSWORD_FILE BEHAVIOUR
BEHAVIOUR: ok | reject_reset (PUT reset-password -> 403) |
           break_new_login (accepts the reset, but only the original password
           ever logs in).
The admin user's current password lives in PASSWORD_FILE.
"""
import json
import sys
import urllib.parse
from http.server import BaseHTTPRequestHandler, HTTPServer

PORT, PWFILE, MODE = int(sys.argv[1]), sys.argv[2], sys.argv[3]
ORIGINAL = open(PWFILE).read()


def current():
    return open(PWFILE).read()


class H(BaseHTTPRequestHandler):
    def reply(self, code, body=None):
        self.send_response(code)
        self.send_header("Content-Type", "application/json")
        self.end_headers()
        if body is not None:
            self.wfile.write(json.dumps(body).encode())

    def do_GET(self):
        if self.path == "/health":
            return self.reply(200, {})
        if self.path.startswith("/admin/realms/master/users?"):
            q = urllib.parse.parse_qs(self.path.split("?", 1)[1])
            if self.headers.get("Authorization") != "Bearer tok" or q.get("username") != ["admin"]:
                return self.reply(401, {"error": "unauthorized"})
            return self.reply(200, [{"id": "u-1", "username": "admin"}])
        self.reply(404, {})

    def do_POST(self):
        n = int(self.headers.get("Content-Length", 0))
        form = urllib.parse.parse_qs(self.rfile.read(n).decode())
        pw = form.get("password", [""])[0]
        ok = form.get("username") == ["admin"] and (pw == current() if MODE != "break_new_login" else pw == ORIGINAL)
        if ok:
            return self.reply(200, {"access_token": "tok"})
        self.reply(401, {"error": "invalid_grant", "error_description": "Invalid user credentials"})

    def do_PUT(self):
        n = int(self.headers.get("Content-Length", 0))
        body = json.loads(self.rfile.read(n) or b"{}")
        if self.headers.get("Authorization") != "Bearer tok" or self.path != "/admin/realms/master/users/u-1/reset-password":
            return self.reply(401, {})
        if MODE == "reject_reset" or body.get("temporary") is not False:
            return self.reply(403, {"error": "refused"})
        open(PWFILE, "w").write(body["value"])
        self.reply(204)

    def log_message(self, *a):
        pass


HTTPServer(("127.0.0.1", PORT), H).serve_forever()
