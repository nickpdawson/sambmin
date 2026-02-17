#!/usr/bin/env python3
"""User operations via samba-tool.

Usage:
    python3 user_ops.py list [--filter=<ldap_filter>]
    python3 user_ops.py create < input.json
    python3 user_ops.py delete <username>
    python3 user_ops.py enable <username>
    python3 user_ops.py disable <username>
    python3 user_ops.py reset-password <username> < input.json
    python3 user_ops.py unlock <username>

All output is JSON to stdout. Errors go to stderr with non-zero exit.
"""

import json
import subprocess
import sys


def run_samba_tool(*args):
    """Execute samba-tool and return stdout."""
    cmd = ["samba-tool"] + list(args)
    result = subprocess.run(
        cmd,
        capture_output=True,
        text=True,
    )
    if result.returncode != 0:
        print(result.stderr, file=sys.stderr)
        sys.exit(result.returncode)
    return result.stdout


def list_users(ldap_filter=None):
    """List all users."""
    args = ["user", "list"]
    if ldap_filter:
        args.extend(["--filter", ldap_filter])
    output = run_samba_tool(*args)
    users = [u.strip() for u in output.strip().split("\n") if u.strip()]
    json.dump({"users": users}, sys.stdout)


def create_user():
    """Create a user from JSON input on stdin."""
    data = json.load(sys.stdin)
    username = data["username"]
    password = data["password"]

    args = ["user", "create", username, password]

    if data.get("givenName"):
        args.extend(["--given-name", data["givenName"]])
    if data.get("surname"):
        args.extend(["--surname", data["surname"]])
    if data.get("mail"):
        args.extend(["--mail-address", data["mail"]])
    if data.get("department"):
        args.extend(["--department", data["department"]])
    if data.get("title"):
        args.extend(["--job-title", data["title"]])
    if data.get("ou"):
        args.extend(["--userou", data["ou"]])
    if data.get("mustChangePassword"):
        args.append("--must-change-at-next-login")

    run_samba_tool(*args)
    json.dump({"success": True, "username": username}, sys.stdout)


def delete_user(username):
    """Delete a user."""
    run_samba_tool("user", "delete", username)
    json.dump({"success": True, "username": username}, sys.stdout)


def enable_user(username):
    """Enable a user account."""
    run_samba_tool("user", "enable", username)
    json.dump({"success": True, "username": username}, sys.stdout)


def disable_user(username):
    """Disable a user account."""
    run_samba_tool("user", "disable", username)
    json.dump({"success": True, "username": username}, sys.stdout)


def reset_password(username):
    """Reset user password from JSON input on stdin."""
    data = json.load(sys.stdin)
    password = data["password"]
    args = ["user", "setpassword", username, f"--newpassword={password}"]
    if data.get("mustChangeAtNextLogin"):
        args.append("--must-change-at-next-login")
    run_samba_tool(*args)
    json.dump({"success": True, "username": username}, sys.stdout)


def unlock_user(username):
    """Unlock a locked user account."""
    run_samba_tool("user", "unlock", username)
    json.dump({"success": True, "username": username}, sys.stdout)


def main():
    if len(sys.argv) < 2:
        print("Usage: user_ops.py <action> [args]", file=sys.stderr)
        sys.exit(1)

    action = sys.argv[1]
    actions = {
        "list": lambda: list_users(
            sys.argv[2].split("=", 1)[1] if len(sys.argv) > 2 and sys.argv[2].startswith("--filter=") else None
        ),
        "create": create_user,
        "delete": lambda: delete_user(sys.argv[2]),
        "enable": lambda: enable_user(sys.argv[2]),
        "disable": lambda: disable_user(sys.argv[2]),
        "reset-password": lambda: reset_password(sys.argv[2]),
        "unlock": lambda: unlock_user(sys.argv[2]),
    }

    if action not in actions:
        print(f"Unknown action: {action}", file=sys.stderr)
        sys.exit(1)

    actions[action]()


if __name__ == "__main__":
    main()
