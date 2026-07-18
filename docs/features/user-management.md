# User Management

The **Users** page lists directory user accounts and is the entry point for the
full account lifecycle. Most actions live in the row's action menu or inside the
user detail drawer (click a user to open it).

> **Who can use it:** creating, modifying, moving, and deleting users are
> **Operator**-level actions (Account Operators / Domain Admins). Any
> authenticated user can view and can change their own password via
> Self-Service.

Writes go through `samba-tool` and direct LDAP modifies as the *logged-in user*,
so every change is attributed to the real actor, not a shared service account.

---

## Creating a user

Open **Create User**. Fields:

- **First / last name** — entering both auto-fills the username, display name,
  and email. The email and the username's `@domain` suffix use your **actual
  domain** (derived from the base DN), not a placeholder.
- **Username** (sAMAccountName), **password** (with a generator), and
  **must change at next login**.
- **Organizational Unit** — create the account directly in any OU. Leave blank
  for the default Users container.
- **Additional groups** — memberships added right after creation. If any group
  add fails (e.g. a name that can't be resolved), the user is still created and
  the UI reports which memberships failed.

If the domain already uses RFC2307 (POSIX) attributes, new users are
automatically assigned `uidNumber`, `gidNumber`, `unixHomeDirectory`, and
`loginShell`. See [Configuration](../CONFIGURATION.md) for the `rfc2307` block.

---

## Editing a user

Open the user drawer. The **Identity**, **Organization**, and **Profile** tabs
hold inline-editable fields (display name, contact details, address, Windows
profile paths, and Unix/POSIX attributes). Click a value to edit; it saves on
blur via an LDAP modify.

---

## The Account tab

The **Account** tab shows account state and the controls for it:

- **Status** — enabled / disabled / locked.
- **Password last set** and **Password expired** (must change at next login).
- **Password Never Expires** — a one-click toggle. When on, the password does
  not age out under the domain's maximum-password-age policy. This is the
  canonical setting for **service accounts**. (Internally it flips the
  `DONT_EXPIRE_PASSWORD` bit in `userAccountControl`.)
- **Account Expires** — click the edit icon to pick an expiration date, or set
  **Never expires**. When an account's expiration date passes, AD **disables**
  the account (re-enable it later with the enable action). Expiry is
  day-granular.

> **Password never expires vs. account expires** — two different things. The
> first is about the *password* aging out (leave a service-account password
> alone); the second is about the whole *account* becoming unusable after a
> date (useful for contractors/temps). A service account typically wants
> password-never-expires **on** and account-expiry set to **never**.

---

## Other lifecycle actions

From the row menu or the drawer:

- **Reset password** — set a new password, optionally forcing a change at next
  login.
- **Enable / Disable / Unlock** — toggle account state; clear a lockout.
- **Rename** — change the account's name (CN/RDN).
- **Move to OU** — relocate the account. The destination picker excludes the
  current parent; the move is by `samba-tool user move`.
- **Group membership** — the drawer's Groups tab adds/removes the user's group
  memberships.
- **Delete** — remove the account (confirmation required).

> **Under the hood:** member and move targets resolve by **sAMAccountName**, not
> the DN's CN (which is usually the display name). Sambmin handles this
> resolution for you — see
> [ARCHITECTURE.md → samba-tool Integration Notes](../ARCHITECTURE.md#samba-tool-integration-notes).

---

## Granting a user rights

To let an account *do* things in the directory — reset others' passwords, create
computers, read a subtree, replicate — use
[Delegation of Control](delegation.md), not group membership alone. Adding an
account to a privileged group is the blunt instrument; delegation grants exactly
the scoped rights it needs.
