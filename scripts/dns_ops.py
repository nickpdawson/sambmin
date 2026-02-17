#!/usr/bin/env python3
"""DNS operations via samba-tool dns and BIND9 nsupdate.

Supports both Samba internal DNS and BIND9 DLZ backends.

Usage:
    python3 dns_ops.py list-zones [--server=<dc>]
    python3 dns_ops.py list-records <zone> [--server=<dc>] [--type=<type>]
    python3 dns_ops.py add-record <zone> < input.json
    python3 dns_ops.py update-record <zone> < input.json
    python3 dns_ops.py delete-record <zone> < input.json
    python3 dns_ops.py create-zone < input.json
    python3 dns_ops.py delete-zone <zone> [--server=<dc>]
    python3 dns_ops.py diagnostics <zone> [--server=<dc>]

All output is JSON to stdout. Errors go to stderr with non-zero exit.
"""

import json
import subprocess
import sys


def run_samba_tool(*args, server=None):
    """Execute samba-tool dns command."""
    cmd = ["samba-tool", "dns"] + list(args)
    if server:
        cmd.extend(["-S", server])
    result = subprocess.run(cmd, capture_output=True, text=True)
    if result.returncode != 0:
        print(result.stderr, file=sys.stderr)
        sys.exit(result.returncode)
    return result.stdout


def run_nsupdate(commands):
    """Execute nsupdate for BIND9 dynamic updates."""
    input_text = "\n".join(commands) + "\nsend\nquit\n"
    result = subprocess.run(
        ["nsupdate", "-g"],  # -g for GSS-TSIG (Kerberos) auth
        input=input_text,
        capture_output=True,
        text=True,
    )
    if result.returncode != 0:
        print(result.stderr, file=sys.stderr)
        sys.exit(result.returncode)
    return result.stdout


def parse_arg(prefix):
    """Extract a --key=value argument."""
    for arg in sys.argv:
        if arg.startswith(f"--{prefix}="):
            return arg.split("=", 1)[1]
    return None


def list_zones():
    server = parse_arg("server")
    output = run_samba_tool("zonelist", server or "localhost", server=server)
    # Parse samba-tool dns zonelist output
    zones = []
    for line in output.strip().split("\n"):
        line = line.strip()
        if line and not line.startswith("pszZoneName"):
            # Basic parsing — will be refined as we test against real output
            zones.append({"name": line})
    json.dump({"zones": zones}, sys.stdout)


def list_records():
    if len(sys.argv) < 3:
        print("Usage: dns_ops.py list-records <zone>", file=sys.stderr)
        sys.exit(1)
    zone = sys.argv[2]
    server = parse_arg("server")
    record_type = parse_arg("type")
    output = run_samba_tool("query", server or "localhost", zone, "@", "ALL", server=server)
    # Parse output — placeholder, needs real Samba output testing
    json.dump({"zone": zone, "records": [], "raw": output}, sys.stdout)


def add_record():
    if len(sys.argv) < 3:
        print("Usage: dns_ops.py add-record <zone> < input.json", file=sys.stderr)
        sys.exit(1)
    zone = sys.argv[2]
    data = json.load(sys.stdin)
    server = data.get("server", "localhost")
    name = data["name"]
    rtype = data["type"]
    value = data["value"]
    run_samba_tool("add", server, zone, name, rtype, value)
    json.dump({"success": True, "zone": zone, "name": name, "type": rtype}, sys.stdout)


def delete_record():
    if len(sys.argv) < 3:
        print("Usage: dns_ops.py delete-record <zone> < input.json", file=sys.stderr)
        sys.exit(1)
    zone = sys.argv[2]
    data = json.load(sys.stdin)
    server = data.get("server", "localhost")
    name = data["name"]
    rtype = data["type"]
    value = data["value"]
    run_samba_tool("delete", server, zone, name, rtype, value)
    json.dump({"success": True, "zone": zone, "name": name, "type": rtype}, sys.stdout)


def main():
    if len(sys.argv) < 2:
        print("Usage: dns_ops.py <action> [args]", file=sys.stderr)
        sys.exit(1)

    action = sys.argv[1]
    actions = {
        "list-zones": list_zones,
        "list-records": list_records,
        "add-record": add_record,
        "delete-record": delete_record,
    }

    if action not in actions:
        print(f"Unknown action: {action}", file=sys.stderr)
        sys.exit(1)

    actions[action]()


if __name__ == "__main__":
    main()
