# Mail platform

## Scope

This application is the GitOps foundation for the Truxonline mail migration from
fuu. It deliberately does **not** deploy a mail server yet.

The active source remains fuu. No MTA, IMAP, webmail, Maildir PVC, public Service,
Ingress, DNS/MX, UDM or Freebox change is declared here.

## Foundation resources

- `mail` namespace with restricted pod-security policy.
- immutable non-secret contract ConfigMap: `truxonline.com`, Maildir, source `fuu`,
  stable mail UID/GID `5000`.
- four OpenBao-backed ExternalSecrets: `mail-auth`, `mail-dkim`, `mail-roundcube`,
  `mail-gmail-import`.
- a visible activation gate ConfigMap.

The OpenBao paths are:

```text
kv/vixens/prod/apps/60-services/mail/auth
kv/vixens/prod/apps/60-services/mail/dkim
kv/vixens/prod/apps/60-services/mail/roundcube
kv/vixens/prod/apps/60-services/mail/gmail-import
```

Placeholders are intentionally present until the read-only fuu audit copies
source values server-to-server. Never commit their values.

## Target architecture after explicit activation

```text
Postfix (SMTP/submission) -> Rspamd -> LMTP -> Dovecot/Pigeonhole -> Maildir
Roundcube -> IMAP + SMTP submission + ManageSieve
Gmail import CronJob -> authenticated SMTP -> Sieve -> Maildir
```

The Maildir PVC will use `truenas-iscsi-xfs-retain` and remain single-writer.
The source migration will use `doveadm sync`, not a live raw filesystem copy.

## Activation gates

1. Verify fuu SSH host identity and inventory Postfix, Dovecot, LDAP, Maildir,
   Roundcube database, DKIM and volumes read-only.
2. Replace every OpenBao placeholder with audited source or newly generated value.
3. Review the image/config manifests and validate the Maildir UID/GID contract.
4. Prove parallel Dovecot Maildir synchronization and rollback.
5. Obtain explicit approval for LoadBalancer ports, UDM/Freebox forwarding and DNS/MX cutover.
