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
