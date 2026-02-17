#!/usr/bin/env python3
"""Group operations via samba-tool.

Usage:
    python3 group_ops.py list
    python3 group_ops.py create < input.json
    python3 group_ops.py delete <groupname>
    python3 group_ops.py add-members <groupname> < input.json
    python3 group_ops.py remove-members <groupname> < input.json
    python3 group_ops.py list-members <groupname>

All output is JSON to stdout. Errors go to stderr with non-zero exit.
"""

import json
import subprocess
import sys


def run_samba_tool(*args):
    cmd = ["samba-tool"] + list(args)
    result = subprocess.run(cmd, capture_output=True, text=True)
    if result.returncode != 0:
        print(result.stderr, file=sys.stderr)
        sys.exit(result.returncode)
    return result.stdout


def list_groups():
    output = run_samba_tool("group", "list")
    groups = [g.strip() for g in output.strip().split("\n") if g.strip()]
    json.dump({"groups": groups}, sys.stdout)


def create_group():
    data = json.load(sys.stdin)
    name = data["name"]
    args = ["group", "add", name]
    if data.get("description"):
        args.extend(["--description", data["description"]])
    if data.get("groupType"):
        args.extend(["--group-type", data["groupType"]])
    if data.get("ou"):
        args.extend(["--groupou", data["ou"]])
    run_samba_tool(*args)
    json.dump({"success": True, "name": name}, sys.stdout)


def delete_group(name):
    run_samba_tool("group", "delete", name)
    json.dump({"success": True, "name": name}, sys.stdout)


def add_members(groupname):
    data = json.load(sys.stdin)
    members = data["members"]  # list of usernames
    for member in members:
        run_samba_tool("group", "addmembers", groupname, member)
    json.dump({"success": True, "group": groupname, "added": members}, sys.stdout)


def remove_members(groupname):
    data = json.load(sys.stdin)
    members = data["members"]
    for member in members:
        run_samba_tool("group", "removemembers", groupname, member)
    json.dump({"success": True, "group": groupname, "removed": members}, sys.stdout)


def list_members(groupname):
    output = run_samba_tool("group", "listmembers", groupname)
    members = [m.strip() for m in output.strip().split("\n") if m.strip()]
    json.dump({"group": groupname, "members": members}, sys.stdout)


def main():
    if len(sys.argv) < 2:
        print("Usage: group_ops.py <action> [args]", file=sys.stderr)
        sys.exit(1)

    action = sys.argv[1]
    actions = {
        "list": list_groups,
        "create": create_group,
        "delete": lambda: delete_group(sys.argv[2]),
        "add-members": lambda: add_members(sys.argv[2]),
        "remove-members": lambda: remove_members(sys.argv[2]),
        "list-members": lambda: list_members(sys.argv[2]),
    }

    if action not in actions:
        print(f"Unknown action: {action}", file=sys.stderr)
        sys.exit(1)

    actions[action]()


if __name__ == "__main__":
    main()
