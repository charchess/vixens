# Runbook — migration mail fuu → Vixens

## Purpose and non-goals

Migrate the Truxonline mail runtime from `fuu` into Vixens without data loss or public-mail interruption. The source remains authoritative until the final approved cutover.

This runbook does **not** authorize a public endpoint, Freebox/UDM change, DNS/MX change, deletion on fuu, or recovery-data removal by itself.

## Verified starting state

- Source: Docker Mailserver v15.1.0, Postfix/Dovecot LMTP + Pigeonhole/ManageSieve, LDAP, OpenDKIM/OpenDMARC, and a separate Roundcube container.
- Source Maildir: `/opt/mailserver/mail-data` → container `/var/mail`; 1,112,392 KiB at audit; UID/GID `5000:5000`; 145 `cur/new/tmp` Maildir component sets.
- Source state: `/opt/mailserver/mail-state` → `/var/mail-state`, root-owned.
- Source queue: empty at audit.
- Target: `mail` Argo application at `ac39eb944c23d35e0f918cf829e107a282b97d06`, `Synced/Healthy`; mailserver and Roundcube templates are at zero replicas; no Pod or PVC exists.
- Target storage: `truenas-iscsi-xfs-retain`, `Retain`, `WaitForFirstConsumer`.
- Target secrets/config: sourced from OpenBao and materialized by eight ready ExternalSecrets.

## Migration mechanism

A dedicated one-shot migration Job will use the pinned image:

```text
instrumentisto/rsync-ssh@sha256:ee4e9665a70742d526cea53d5d5b012990a39fbb4f87b7f094250e44ab93db5f
```

It mounts the target Maildir claim and **pulls** `/opt/mailserver/mail-data/` from fuu through a dedicated, restricted, revocable SSH key. The DMS image is deliberately not used for this Job because it has neither `rsync` nor an SSH client.

The Job must use `rsync -aHAX --numeric-ids` and an SSH command with `StrictHostKeyChecking=yes` plus a pinned fuu host key. The initial and intermediate passes do **not** delete target paths. Only the final pass, after the source writer has been quiesced, may add `--delete-delay`.

## Stage A — implementation review gate

Required GitOps change, still inactive:

1. Add a CiliumNetworkPolicy selecting only `app.kubernetes.io/part-of=truxonline-mail` migration pods, permitting egress only to fuu TCP/22 and required DNS.
2. Add a suspended migration Job with the pinned image, target PVC mount, source host-key Secret mount, migration-key Secret mount, and explicit non-deleting initial command.
3. Add a target Certificate for `mail.truxonline.com` in namespace `mail`; it may use DNS-01 but does not publish a mail endpoint.
4. Add an internal-only test access path, never the production hostname/ports.
5. Render/lint the mail and parent Argo overlays. Merge/promotion is a separate approval gate.

Acceptance: no public LoadBalancer/TCPRoute/Ingress, no port forwarding, no MX/DNS cutover, no source modification.

## Stage B — storage and isolated runtime gate

Requires explicit approval because it creates a TrueNAS volume and starts a private target workload.

1. Set the DMS StatefulSet to one replica only after its volume claim binds through `truenas-iscsi-xfs-retain`.
2. Verify the PV uses the intended TrueNAS CSI provisioner and `Retain` reclaim policy.
3. Verify target mount ownership can represent `/var/mail` as `5000:5000` and `/var/mail-state` as root-owned.
4. Do not expose SMTP, IMAP, submission, Roundcube, or ManageSieve beyond the cluster.
5. Verify DMS starts with its mounted configuration and certificate, but performs no public mail intake.

Rollback: scale the target back to zero; retain the bound PVC; fuu remains untouched.

## Stage C — initial and incremental data synchronization gate

Requires explicit approval to install the dedicated restricted public key on fuu's root account (or an equivalent read-only migration account). This changes fuu access control but does not change mail traffic.

1. Generate one dedicated Ed25519 migration key, store its private part only in OpenBao, and add only its public key to fuu with a restrictive `authorized_keys` command/forwarding policy.
2. Pin fuu's existing SSH host fingerprint in the target known-hosts Secret.
3. Run the initial migration Job with no delete flag.
4. Record only aggregate source/target counts, total bytes, rsync exit status, ownership/mode checks, and sampled Maildir structure. Do not log mail paths, message IDs, addresses, or content.
5. Repeat non-deleting rsync deltas while fuu remains authoritative.

Acceptance: migration Job exit code 0; target filesystem passes ownership and Maildir shape checks; aggregate counts and bytes are within the documented expected delta.

Rollback: delete/scale down only the migration Job; keep target PVC retained; revoke the dedicated source key if stopping the migration effort.

## Stage D — private protocol validation gate

Requires explicit approval to start DMS and Roundcube privately.

1. Validate IMAPS, submission STARTTLS, SMTP relay restrictions, LMTP delivery, and ManageSieve from a controlled in-cluster client.
2. Validate an existing test mailbox end-to-end without delivering external production mail through Vixens.
3. Validate DKIM signing with a test message; preserve the same selector/domain contract until a later deliberate rotation.
4. Validate Roundcube through an internal-only route; it must use the target DMS, not fuu.
5. Keep Gmail import disabled. It resumes only after the authoritative path is established and Sieve behavior is validated.

Acceptance: all private protocol tests pass; source fuu traffic is unaffected; Argo reports the intended revision and healthy workloads.

Rollback: scale target workloads to zero; source remains authoritative; preserve the retained target PVC for diagnosis.

## Stage E — final cutover gate

Requires explicit approval for a scheduled maintenance window and the public traffic changes.

Preconditions:

- successful Stage C/D evidence;
- current backups/snapshots for fuu mail data and target TrueNAS volumes;
- explicit Freebox → UDM → Vixens TCP mapping prepared for ports 25, 465, 587, 993, and optionally 4190 if ManageSieve is intentionally public;
- Cilium LoadBalancer TCP service, NetworkPolicy, Certificate, SPF/DKIM/DMARC, MX, and reverse-DNS plan reviewed;
- rollback owner and maximum rollback interval named.

Execution:

1. Stop/quiesce the source DMS writer on fuu during the approved window.
2. Run the final rsync delta with `--delete-delay`; require success.
3. Start target DMS and verify private local listeners and mail queue.
4. Change public TCP forwarding to Vixens; do not remove fuu configuration or data.
5. Test external inbound SMTP, outbound authenticated submission, IMAPS, Sieve, DKIM, Roundcube, and rejected-relay behavior.
6. Change MX/DNS only where it is actually required by the verified forwarding design; verify public DNS using DoH/external resolution, not LAN split DNS alone.
7. Keep source fuu stopped but recoverable for the agreed observation window.

Immediate rollback:

1. Restore Freebox/UDM forwarding to fuu.
2. Stop Vixens mail workloads and leave target PVC untouched.
3. Start fuu DMS using its existing volumes/configuration.
4. Validate external SMTP and IMAPS against fuu.
5. Investigate from preserved target volume and Job logs; do not reverse-copy onto fuu without a separate approval.

## Completion criteria

Migration is complete only after the observation window passes with:

- Argo `Synced/Healthy` at the intended promoted revision;
- inbound/outbound SMTP, submission, IMAPS, Sieve, and Roundcube functional;
- DKIM/SPF/DMARC and TLS valid externally;
- no queue growth or duplicate delivery anomaly;
- target snapshots/replication proven;
- fuu retained until the owner separately authorizes retirement.
