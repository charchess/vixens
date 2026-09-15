# Mail platform

## Scope

This application migrates the Truxonline mail runtime from fuu into Vixens in controlled stages. fuu is the active authoritative source until the explicit public cutover gate.

The production GitOps stage currently declares pinned Docker Mailserver (DMS) and Roundcube templates at **zero replicas**. No mail Pod or Maildir PVC exists; no public mail listener, TCPRoute/Ingress, DNS/MX, UDM or Freebox change has been declared.

## Live staging contract

- `mail` namespace uses Pod Security `baseline`: DMS needs a root bootstrap phase but receives no privileged container configuration.
- DMS image is pinned to the audited fuu v15.1.0 digest; Roundcube is separately pinned.
- Eight OpenBao-backed ExternalSecrets materialize auth, DKIM, DMS config/runtime, Roundcube config/runtime, and Gmail-import contracts. Values are never stored in Git.
- DMS and Roundcube are both `replicas: 0`.
- The two Services are `ClusterIP` only; they are not public endpoints.
- Maildir and state claims are StatefulSet templates using `truenas-iscsi-xfs-retain`; `WaitForFirstConsumer` means no PVC is instantiated while replicas remain zero.

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

The DMS StatefulSet runs at one private replica only after TLS and secret contracts are verified. Roundcube remains at zero replicas and the migration Job remains suspended until its separate sync gate. No public mail routing, DNS, MX, NAT, or TCP exposure is declared by this stage.


### DMS capability boundary

The DMS image uses `supervisord` and must `chown` only its container-local `/dev/shm` control socket. The mailserver container therefore explicitly drops every Linux capability except `CHOWN`, while retaining `allowPrivilegeEscalation: false`. This is a workload-level Linux capability, not Talos node access; it grants neither host mounts nor host namespaces nor privileged execution.


### Runtime capabilities for DMS private validation

DMS needs a limited set of workload-local Linux capabilities to supervise Postfix/Dovecot, switch service UIDs/GIDs, manage its own socket/files, bind the mail ports, and use Postfix chroot. The explicit allowlist is `CHOWN`, `DAC_OVERRIDE`, `FOWNER`, `MKNOD`, `SETGID`, `SETUID`, `NET_BIND_SERVICE`, `SYS_CHROOT`, and `KILL`; all others are dropped. `NET_ADMIN` and `NET_RAW` remain absent because Fail2Ban is disabled. The container remains non-privileged with `allowPrivilegeEscalation: false` and `RuntimeDefault` seccomp; this grants no Talos node or host-namespace access.


## Initial Maildir sync

The initial migration Job is enabled for one non-destructive, non-deleting `rsync -aHAX --numeric-ids --partial` pull from fuu. It is intentionally single-completion with `backoffLimit: 0`; it does not alter fuu, public routing, or the final-cutover procedure. A final delta with deletion remains a separate explicitly approved window.


### Initial sync ownership contract

The initial rsync runs as UID/GID 0 with `fsGroup: 5000` to access the target Maildir root (`root:5000`, mode `2775`). It has only `CHOWN`, `FOWNER`, and `DAC_OVERRIDE` to preserve source Maildir numeric ownership and timestamps. It retains `drop: [ALL]`, `allowPrivilegeEscalation: false`, and `RuntimeDefault` seccomp. The failed v1 Job transferred zero bytes and is replaced by the v2 Job because Kubernetes Job templates are immutable.
