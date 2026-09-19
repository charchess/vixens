# Mail platform

## Scope

This application migrates the Truxonline mail runtime from fuu into Vixens in controlled stages. fuu is the active authoritative source until the explicit public cutover gate.

The production GitOps stage declares a pinned Docker Mailserver (DMS) and Roundcube at **one replica each** for private functional validation. The Maildir and state PVCs are retained TrueNAS claims populated by the non-destructive initial sync. The existing HTTPS webmail Ingress is used for this validation; no mail-protocol public listener, DNS/MX, UDM or Freebox change is declared.

## Live staging contract

- `mail` namespace uses Pod Security `baseline`: DMS needs a root bootstrap phase but receives no privileged container configuration.
- DMS image is pinned to the audited fuu v15.1.0 digest; Roundcube is separately pinned.
- Eight OpenBao-backed ExternalSecrets materialize auth, DKIM, DMS config/runtime, Roundcube config/runtime, and Gmail-import contracts. Values are never stored in Git.
- DMS and Roundcube are each `replicas: 1` for private validation.
- The two Services are `ClusterIP` only; they are not public endpoints.
- Maildir and state claims use `truenas-iscsi-xfs-retain`; the retained claims are Bound after the initial sync and attach only to the one private DMS replica during validation.

## Migration model

The source is a DMS v15.1.0 stack with Postfix/Dovecot LMTP + Pigeonhole/ManageSieve, LDAP, OpenDKIM/OpenDMARC, Maildir, and separate Roundcube. The first Vixens migration preserves this server/client boundary and the portable Maildir format. Rspamd extraction remains a later, independent hardening migration rather than a simultaneous data-platform rewrite.

The dedicated migration Job uses a pinned rsync+SSH image and pulls data from fuu through a restricted, revocable migration key. It is not part of the DMS image and is introduced only after separate approval.

## Runbook and gates

See [mail migration runbook](../../runbooks/60-services/mail-migration.md).

Before any target workload runs, the following remain mandatory:

1. reviewed Cilium egress policy and a restricted temporary source SSH identity;
2. bound TrueNAS claims plus isolated DMS mount/ownership validation;
3. non-deleting incremental transfer and aggregate parity evidence;
4. private SMTP/IMAP/Sieve/Roundcube/DKIM validation;
5. explicit scheduled approval for public TCP forwarding and any DNS/MX cutover.

A successful ExternalSecret or Git promotion is not activation proof.


## Private DMS validation stage

The DMS StatefulSet and Roundcube Deployment run at one replica each after TLS and secret contracts are verified. The migration Job remains suspended until its separate sync gate. No public mail-protocol routing, DNS, MX, NAT, or TCP exposure is declared by this stage; the pre-existing HTTPS webmail Ingress is used only for the authorized validation.


### DMS capability boundary

The DMS image uses `supervisord` and must `chown` only its container-local `/dev/shm` control socket. The mailserver container therefore explicitly drops every Linux capability except `CHOWN`, while retaining `allowPrivilegeEscalation: false`. This is a workload-level Linux capability, not Talos node access; it grants neither host mounts nor host namespaces nor privileged execution.


### Runtime capabilities for DMS private validation

DMS needs a limited set of workload-local Linux capabilities to supervise Postfix/Dovecot, switch service UIDs/GIDs, manage its own socket/files, bind the mail ports, and use Postfix chroot. The explicit allowlist is `CHOWN`, `DAC_OVERRIDE`, `FOWNER`, `MKNOD`, `SETGID`, `SETUID`, `NET_BIND_SERVICE`, `SYS_CHROOT`, and `KILL`; all others are dropped. `NET_ADMIN` and `NET_RAW` remain absent because Fail2Ban is disabled. The container remains non-privileged with `allowPrivilegeEscalation: false` and `RuntimeDefault` seccomp; this grants no Talos node or host-namespace access.


## Initial Maildir sync

The initial migration Job is enabled only after the private DMS and Roundcube workloads are quiesced and the retained RWO Maildir claim is verified free to mount. It performs one non-destructive, non-deleting `rsync -aHA --no-xattrs --numeric-ids --partial` pull from fuu. It is intentionally single-completion with `backoffLimit: 0`; it does not alter fuu, public routing, or the final-cutover procedure. A final delta with deletion remains a separate explicitly approved window.


### Initial sync ownership contract

The initial rsync runs as UID/GID 0 with `fsGroup: 5000` to access the target Maildir root (`root:5000`, mode `2775`). It has only `CHOWN`, `FOWNER`, and `DAC_OVERRIDE` to preserve source Maildir numeric ownership and timestamps. It retains `drop: [ALL]`, `allowPrivilegeEscalation: false`, and `RuntimeDefault` seccomp. The failed v1 Job transferred zero bytes and is replaced by the v2 Job because Kubernetes Job templates are immutable.


### SELinux xattrs excluded from portable Maildir sync

The v2 initial transfer copied the Maildir payload but ended with rsync exit 23 because source `security.selinux` xattrs cannot be written to the target iSCSI XFS volume under Talos. These labels are source-host policy metadata, not portable mail data. The v3 Job deliberately uses `--no-xattrs` while retaining archive mode, hard links, ACL preservation, numeric IDs, and all non-destructive constraints.


### Private LDAP and Postfix map readiness

DMS authentication is constrained by a Cilium policy to the two audited LDAP endpoints (`192.168.200.21` and `.22`, TCP/389) plus Kubernetes CoreDNS TCP/UDP 53. It exposes no public mail route. The DMS config init generates `postfix-virtual.cf.db` through idempotent `postmap` after materializing the source map, fixing the required hash map without placing generated data in Git.

### Bounded Active Directory search base

All mail users are administered under `OU=users,OU=infra,DC=internal,DC=truxonline,DC=com`. `LDAP_SEARCH_BASE` is explicitly limited to that OU rather than the AD domain root. This prevents Dovecot/libldap from following unrelated AD partition referrals during passdb lookup while retaining the complete authorized mail-user population.

### Dovecot LDAP authentication blocker

Private validation established that the service-account bind and the exact LDAP search complete in approximately 100 ms from the DMS Pod against both DCs. The effective fuu DMS source configuration was then read through its approved operator path: its Dovecot LDAP URI list is `ldap://192.168.200.21 ldap://192.168.200.22`, while the target had only DC1. The DMS template did not derive `DOVECOT_URIS` from target `LDAP_SERVER_HOST` and retained `ldap://mail.example.com`; runtime now declares `DOVECOT_URIS` explicitly at the same DC1+DC2 value. Dovecot/libldap later demonstrated an AD-domain-root referral hang despite successful raw LDAP queries, so the authorized mail-user OU is now the explicit `LDAP_SEARCH_BASE`. The validation gate is a prompt normal `doveadm` refusal for an unknown user and a real Roundcube login; no passdb semantic rewrite is required.


### Temporary private LAN Roundcube validation

The production overlay may temporarily run one Roundcube replica behind `roundcube-private-lan` NodePort `30443`. A non-root TLS proxy sidecar terminates the existing `mail.truxonline.com` certificate and permits only `192.168.199.0/24` through the selected Pod's Cilium ingress policy. `externalTrafficPolicy: Local` preserves client source addresses; use the Node IP hosting the Roundcube Pod. This is a test-only endpoint and must be removed after the owner validates login and mailbox visibility. No DNS, MX, NAT, or public route is changed.


The private LAN Roundcube pod uses only workload-local bootstrap capabilities (`CHOWN`, `DAC_OVERRIDE`, `FOWNER`, `SETGID`, `SETUID`) because the official Apache image creates its ephemeral SQLite state and drops to `www-data`. Its `/var/roundcube` and `/tmp/roundcube-temp` volumes are `emptyDir`: this test endpoint intentionally retains no application state after removal.


### Authorized webmail reverse-proxy cutover

`mail.truxonline.com` continues to terminate TLS at the existing Traefik `mail-gateway` Ingress, but its former static HTTP endpoint was replaced by the internal `mail/roundcube` ClusterIP (`10.107.97.165:80`). The temporary NodePort/TLS sidecar path was removed. Cilium admits only the selected Traefik Pods to the Roundcube backend TCP/80. SMTP, IMAP, MX, DNS, NAT and fuu authority remain outside this webmail-only change.

### Roundcube authenticated Submission

Roundcube submits through `tls://mailserver.mail.svc.cluster.local:587` using its authenticated session placeholders (`%u` and `%p`) with `PLAIN` after STARTTLS. TLS peer and hostname validation remain enabled; the connection explicitly verifies the issued certificate name `mail.truxonline.com` while routing to the internal Service FQDN. This keeps SMTP within the mail namespace and validates the target Postfix Submission path without exposing a public SMTP endpoint or changing MX/NAT.

The Docker runtime variables use the same internal STARTTLS endpoint and port. This prevents the image-generated runtime configuration from falling back to the legacy `mailserver:25` endpoint after startup.

### Gmail IMAP import with explicit EXPUNGE

The `gmail-import` CronJob drains `[Gmail]/All Mail` every ten minutes, in batches of 50. It therefore includes archived/label-only Gmail messages as well as those carrying INBOX. It fetches each RFC822 message over IMAPS from `imap.gmail.com`, delivers it to the local DMS through the in-cluster SMTP Service, then marks that exact Gmail UID `\\Deleted` and executes `EXPUNGE` only after SMTP acceptance. The user explicitly selected this expunge strategy; it permanently removes imported source messages from Gmail rather than retaining Gmail Trash as a backup. The completed Job record follows the platform-enforced 3600-second TTL. The SMTP transaction uses only the external envelope `gmail-import@truxonline.invalid`, which Postfix accepts without weakening unauthenticated local-domain sender protection; the imported RFC822 headers and payload remain unchanged. A failed SMTP delivery leaves the Gmail message untouched. The selected importer has no Kubernetes API token and Cilium permits only DNS through CoreDNS (for FQDN enforcement), `imap.gmail.com:993`, and `mailserver:25`.

The DMS `user-patches.sh` hook creates `tmp` and `db` with Amavis ownership (`999:999`) and restrictive modes after DMS setup and before Amavis starts. This is required for the localhost content-filter listener on port `10024`; it contains no mailbox data and is deliberately separate from the retained Maildir and mail-state PVCs.
