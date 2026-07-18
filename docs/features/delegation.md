# Delegation of Control

The **Delegation** page (under **Policy & Security**) grants scoped Active
Directory rights on an OU — or the whole domain — to users and groups. It is the
browser equivalent of the Windows "Delegation of Control Wizard" plus the parts
of `dsacl.exe` / `samba-tool dsacl` you would otherwise run by hand.

Use it to stand up the accounts a real environment needs:

- **Service accounts** — e.g. a domain-join account that can create computer
  objects in one OU, and nothing else.
- **Bind / directory-sync accounts** — a read-only account for an app's LDAP
  bind, or a sync account that needs directory-replication rights.
- **OU administrators** — full control over a branch of the tree without
  Domain Admin.
- **Help-desk delegates** — reset passwords for one department's users.

> **Who can use it:** Delegation is an **Admin**-only feature (Domain Admins /
> Enterprise Admins). Granting rights is itself a privileged operation, and
> writing an object's ACL requires `WRITE_DAC`, which the delegate performing
> the grant must hold on the target.

---

## How it works

1. **Pick a target** — an OU, or the domain root. The target defaults to the
   domain root.
2. **Select trustees** — one or more users and/or groups who will *receive* the
   rights. (Delegating to a group is usually better than to individuals — add
   and remove people from the group later without touching ACLs.)
3. **Select capabilities** — one or more templates (see the reference below).
4. **Grant** — every selected trustee is granted every selected capability in a
   single action (the full trustee × capability matrix). High-privilege
   selections ask for confirmation first.

The **Current delegations** panel below the form lists the delegations
*explicitly set* on the selected object, one row per trustee + capability, each
with a **Remove** button. Inherited defaults and built-in ACEs (SYSTEM, Domain
Admins, …) are deliberately hidden — only rights explicitly delegated to real
domain principals are shown.

Delegated rights are **inherited** down the subtree (they use container
inheritance), so a capability granted on an OU applies to matching objects in
that OU and everything beneath it.

---

## Capability reference

Each capability is a small, named bundle of access-control entries (ACEs).
Object delegations are written as SDDL; the two directory-replication rights use
Samba's named control-access-right (`--car`) interface, because that is the only
way Samba exposes them.

| Capability | Category | Risk | What it grants |
|---|---|---|---|
| **Reset user passwords** | User accounts | medium | Reset the password of users in the target, and force a change at next logon. The classic help-desk right. |
| **Create, delete, and manage user accounts** | User accounts | high | Create and delete user objects, and read/write all their properties. Full user lifecycle. |
| **Manage group membership** | Groups | medium | Add and remove members of groups (read/write the `member` attribute). |
| **Create, delete, and manage groups** | Groups | high | Create and delete group objects and read/write all their properties. |
| **Create and delete computer accounts** | Computers | medium | Create and delete computer objects and manage their properties — a domain-join service account. |
| **Read all objects and properties** | Read access | low | Read every object and property in the target subtree. A read-only bind/service account. |
| **Full control** | Full control | high | Complete control over the target and everything under it, including permissions. An OU administrator. |
| **Replicate directory changes** | Directory replication | high | Read directory changes across the domain (DirSync). Apply on the **domain root**. |
| **Replicate directory changes: All** | Directory replication | high | Read all changes **including password hashes and secrets** (e.g. Azure AD Connect / password-hash sync). Apply on the **domain root**. |

<details>
<summary>Exact ACEs applied (for auditors)</summary>

The `%SID%` placeholder is replaced with the trustee's `objectSid`. Well-known
GUIDs are the standard AD schema values (identical in Samba).

| Capability | ACE(s) / CAR |
|---|---|
| Reset user passwords | `(OA;CI;CR;00299570-246d-11d0-a768-00aa006e0529;<user-class>;%SID%)` + `(OA;CI;WP;<pwdLastSet>;<user-class>;%SID%)` |
| Create/delete/manage user accounts | `(OA;CI;CCDC;<user-class>;;%SID%)` + `(OA;CI;RPWP;;<user-class>;%SID%)` |
| Manage group membership | `(OA;CI;RPWP;<member-attr>;<group-class>;%SID%)` |
| Create/delete/manage groups | `(OA;CI;CCDC;<group-class>;;%SID%)` + `(OA;CI;RPWP;;<group-class>;%SID%)` |
| Create/delete computer accounts | `(OA;CI;CCDC;<computer-class>;;%SID%)` + `(OA;CI;RPWP;;<computer-class>;%SID%)` |
| Read all objects and properties | `(A;CI;LCRPRC;;;%SID%)` |
| Full control | `(A;CI;GA;;;%SID%)` |
| Replicate directory changes | `dsacl set --car=get-changes --action=allow --trusteedn=<DN>` |
| Replicate directory changes: All | `dsacl set --car=get-changes-all --action=allow --trusteedn=<DN>` |

`CI` = container-inherit. Class/attribute GUIDs: user `bf967aba-…`, group
`bf967a9c-…`, computer `bf967a86-…`, `member` `bf9679c0-…`, `pwdLastSet`
`bf967a0a-…`.

</details>

---

## Recipes

### Read-only bind account for an app

1. Create a user (e.g. `svc-app-bind`) — see [User Management](user-management.md).
   Set **Password never expires** on its Account tab.
2. On **Delegation**, target the OU whose objects the app needs to read (or the
   domain root for everything), select the bind account, and grant **Read all
   objects and properties**.

### Domain-join service account

1. Create the account (e.g. `svc-join`) and set the password to never expire.
2. Target the OU where computers should land (e.g. `OU=Workstations`), select
   the account, and grant **Create and delete computer accounts**. The account
   can now join machines into that OU and nowhere else.

### Directory-sync / DirSync bind account

1. Create the account and set the password to never expire.
2. Target the **domain root** and grant **Replicate directory changes** (read
   sync) — or **Replicate directory changes: All** if the tool needs password
   hashes / secrets (e.g. Azure AD Connect password-hash sync). The page warns
   you if a replication capability is selected while the target is an OU.

### Help-desk password resets for one department

1. Create a group (e.g. `Helpdesk-Sales`) and add the help-desk users to it.
2. Target `OU=Sales`, select the group, and grant **Reset user passwords**.
   Manage who can reset by editing group membership — no ACL changes needed.

### OU administrator

1. Create a group (e.g. `OU-Admins-Sales`).
2. Target `OU=Sales`, select the group, grant **Full control**.

---

## Removing a delegation

Find the row in **Current delegations** and click **Remove**. A delegation may
be stored as more than one ACE (a "Reset user passwords" grant is two ACEs; a
"Full control" grant is stored as two after AD canonicalizes it) — Remove
deletes the whole group together, so one click cleans it up entirely.

---

## Notes and caveats

- **Inheritance:** capabilities are container-inherited, so they apply to the
  target and its whole subtree. Grant on the narrowest OU that covers what the
  trustee needs.
- **Groups over individuals:** delegate to a group and manage people via
  membership; it keeps ACLs stable and auditable.
- **Replication rights belong on the domain root.** `get-changes` /
  `get-changes-all` only take effect on the domain naming context. The page
  warns when the target is an OU.
- **"Replicate … All" exposes secrets.** It can read password hashes. Treat any
  account holding it as Tier-0 and protect it accordingly.
- **What's shown vs. what exists:** the panel shows only *explicit* delegations
  to domain principals. The object's full ACL (inherited ACEs, class defaults,
  the SACL) is not displayed; use `samba-tool dsacl get --objectdn=<dn>` on a DC
  for the raw descriptor.

For the implementation details (how Sambmin drives `samba-tool dsacl`, SDDL
parsing, generic-rights canonicalization), see
[ARCHITECTURE.md → Delegation of Control](../ARCHITECTURE.md#delegation-of-control-dsacl).
