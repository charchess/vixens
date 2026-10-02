# Actual Budget

## Deployment

- Namespace: `finance`
- URL: `https://actual.truxonline.com`
- Image: `actualbudget/actual-server:sha-5e39105` (immutable upstream revision)
- State: single RWO TrueNAS iSCSI XFS PVC, `actual-budget-data` (5 GiB). All account/session SQLite data and encrypted budget files are under `/data`.

## First bootstrap

After the approved production promotion, immediately visit the TLS URL from a trusted client and set the Actual server password before importing or creating a budget. Do not record this application-local password in Git or OpenBao.

## Backup / restore

Back up the complete PVC via the standard backup profile. For a file-level restore, quiesce the workload first because `/data/server-files/account.sqlite` is SQLite; restore the complete `/data` tree, then start Actual and verify login plus budget sync.
