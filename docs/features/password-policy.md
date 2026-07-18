# Password Policy

The **Password Policies** page (under **Policy & Security**) manages the domain
default password policy, Fine-Grained Password Policies (PSOs), and includes a
password tester.

> **Who can use it:** viewing the policy is available to any authenticated user;
> editing the domain policy and creating/applying PSOs are **Admin**-level
> actions.

---

## Domain default policy

The **Domain Default Policy** tab shows the domain-wide settings and lets an
admin edit them:

- Minimum password length
- Password history length
- Minimum and maximum password age
- Password complexity (on/off)
- Store plaintext passwords (on/off)
- Account lockout threshold, duration, and observation window

Edits are applied via `samba-tool domain passwordsettings set`.

---

## Fine-Grained Password Policies (PSOs)

The **Fine-Grained Policies** tab lists Password Settings Objects, which
override the domain default for specific users or groups. Lower **precedence**
values win. You can:

- **Create** a PSO with its own length/history/age/complexity/lockout settings.
- **Apply** or **remove** a PSO to/from a user or group (by sAMAccountName).
- **Delete** a PSO.

Common use: a stricter PSO applied to a privileged group (e.g. longer minimum
length and shorter max age for Domain Admins).

---

## Password tester

The **Password Tester** tab checks a candidate password against the effective
policy — the domain default, or a user-specific PSO when a username is supplied.
It reports minimum-length and complexity-category failures and whether the
password contains the username. Useful for pre-flighting a password before a
reset.

---

## Notes

- The effective policy for a user is the PSO that applies to them (by direct
  assignment or group), falling back to the domain default. The tester and the
  per-user effective-policy lookup follow that same resolution.
- Complexity in AD means the password must contain characters from at least
  three of: uppercase, lowercase, digits, and non-alphanumeric — and must not
  contain the account name. The tester enforces the same rule client-side.
