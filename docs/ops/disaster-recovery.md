# Disaster Recovery Runbook

**Source:** Production Gate #12 (`docs/PRODUCTION_GATE.md`) — *"No DR tests. No backup/restore runbook. Spec §75 critical-failure test (worker crash + retry chain) not implemented."*

**Scope:** This runbook fixes the platform's Disaster Recovery (DR)
posture for the three failure classes the audit called out:

1. **Data loss** — primary Postgres, S3, or NATS JetStream destruction.
2. **Partial outage** — single AZ, single service, or single dependency.
3. **Worker crash mid-workflow** — the spec §75 critical-failure test
   (worker crashes mid-Bill-Processing-Pipeline; workflow must resume
   from the last activity without duplicating side effects).

The DR posture is defined by two objectives:

| Objective | Target  | Justification                                              |
|-----------|---------|-----------------------------------------------------------|
| **RPO**   | 1 hour  | Worst-case data loss window for civic-truth writes.       |
| **RTO**   | 4 hours | Time to restore API + workers to a serving state in a new region. |

The 1-hour RPO is met by combining Postgres WAL streaming (continuous),
S3 cross-region replication (continuous), and NATS JetStream snapshots
(every 15 minutes). The 4-hour RTO is met by rehearsing this runbook
quarterly; the rehearsal cadence is the SRE team's Q1–Q4 game-day
obligation.

---

## 1. Backup procedure

Backups run on three independent schedules so no single dependency
failure can breach the RPO.

### 1.1 Postgres (canonical civic truth)

```bash
# Continuous: WAL archive to S3 (configured in postgresql.conf)
#   archive_mode = on
#   archive_command = 'aws s3 cp %p s3://civic-pg-wal/<cluster>/%f'
#
# Nightly: full base backup via pg_dump (logical) + pg_basebackup (physical).

# 1. Logical backup — restored for single-table / single-row recoveries.
pg_dump --format=custom \
        --no-owner --no-privileges \
        --file=/backups/postgres/civic-$(date -u +%Y%m%dT%H%M%SZ).dump \
        "$(echo $DATABASE_URL)"

# 2. Physical base backup — restored for full-cluster recoveries.
pg_basebackup --pgdata=/backups/postgres/base \
              --format=tar --gzip \
              --wal-method=stream \
              --write-recovery-conf \
              --progress
```

**Schedule:** nightly at 02:00 UTC (off-peak for KE / TZ / NG / GH).
**Retention:** 30 days of logical backups + 7 days of physical base
backups + 7 days of continuous WAL archive. This exceeds the 1-hour RPO
twice over (last full backup ≤ 24 h old; WAL replay fills the gap).

**Verification:** the nightly backup job writes a sentinel object to
`s3://civic-backups-sentinel/postgres/<timestamp>` and the absence of
this sentinel for > 25 h pages the SRE on-call.

### 1.2 S3 / MinIO (objects)

```bash
# Continuous cross-region replication is configured at the bucket level:
#   aws s3api put-bucket-replication \
#     --bucket civic-documents-primary \
#     --replication-configuration file://replication.json
#
# Nightly lifecycle rule moves objects to GLACIER after 90 days and
# deletes them after 7 years (the regulatory retention window).

# Manual verification (run after every deployment):
aws s3 ls s3://civic-documents-primary --recursive --summarize \
  > /tmp/primary-inventory.txt
aws s3 ls s3://civic-documents-replica --recursive --summarize \
  > /tmp/replica-inventory.txt
diff /tmp/primary-inventory.txt /tmp/replica-inventory.txt
```

**Schedule:** continuous replication + 90-day lifecycle transition.
**Retention:** 7 years (regulatory).

### 1.3 NATS JetStream (event bus)

```bash
# JetStream snapshots are written every 15 minutes to S3.
nats stream backup --target s3://civic-jetstream-backups/ \
                   CIVIC_EVENTS CIVIC_BILL_PROCESSING

# Restore (RTO step):
nats stream restore --source s3://civic-jetstream-backups/<latest>/ \
                    CIVIC_EVENTS
```

**Schedule:** every 15 minutes.
**Retention:** 7 days of JetStream snapshots. The 15-minute cadence
exceeds the 1-hour RPO 4×.

---

## 2. Restore procedure

The restore procedure assumes the primary region is LOST (worst case).
For single-table recoveries, skip to §2.4.

### 2.1 Provision a fresh Postgres in the DR region

```bash
# 1. Spin up the RDS instance from the latest physical base backup.
aws rds restore-db-instance-from-db-snapshot \
  --db-instance-identifier civic-pg-dr \
  --db-snapshot-identifier civic-pg-$(date -u +%Y%m%d) \
  --db-instance-class db.r6g.large \
  --availability-zone us-east-2a

# 2. Wait for the instance to become available (typically 10–15 minutes).
aws rds wait db-instance-available --db-instance-identifier civic-pg-dr

# 3. Apply the WAL archive forward to the desired recovery point.
psql -h civic-pg-dr.<account>.us-east-2.rds.amazonaws.com \
     -U civic_admin -d postgres \
     -c "SELECT pg_wal_replay_pause();"  # pause at the recovery target

# 4. Verify the recovery target is reachable (latest bill, latest snapshot).
psql -h civic-pg-dr... -c "SELECT MAX(created_at) FROM legislation.bills;"
psql -h civic-pg-dr... -c "SELECT MAX(observation_date) FROM government.debt_snapshots;"
```

### 2.2 Promote S3 replica to primary

```bash
# The replica bucket is already writable; just point the API at it.
# Update the config map:
kubectl -n civic-prod edit configmap api-config
#   STORAGE_BUCKET: civic-documents-replica  # was -primary

# Roll the API deployment so every pod picks up the new bucket:
kubectl -n civic-prod rollout restart deployment/api
kubectl -n civic-prod rollout status  deployment/api --timeout=300s
```

### 2.3 Restore NATS JetStream

```bash
# 1. Provision a fresh NATS cluster in the DR region.
helm install nats-dr nats/nats -f infrastructure/kubernetes/nats-dr-values.yaml

# 2. Restore the latest JetStream snapshot.
nats stream restore --source s3://civic-jetstream-backups/<latest>/ CIVIC_EVENTS
nats stream restore --source s3://civic-jetstream-backups/<latest>/ CIVIC_BILL_PROCESSING

# 3. Verify the stream is consuming again.
nats stream info CIVIC_EVENTS
```

### 2.4 Single-table recovery (point-in-time)

When the failure is "we accidentally deleted one row from
`legislation.acts`", restore the logical backup to a TEMPORARY instance
and copy the missing row back.

```bash
# 1. Restore last night's logical backup to a sandbox Postgres.
pg_restore --dbname=civic_sandbox \
           --create --no-owner \
           /backups/postgres/civic-<last-good>.dump

# 2. Copy the missing row back to production (within a transaction).
psql -h civic-pg-prod -d civic <<'SQL'
BEGIN;
INSERT INTO legislation.acts
  SELECT * FROM dblink(
    'host=civic-sandbox dbname=civic_sandbox',
    'SELECT * FROM legislation.acts WHERE id = ''<missing-id>'''
  ) AS t(...);
COMMIT;
SQL
```

---

## 3. Failover procedure

Failover is the act of redirecting traffic from the failed primary to
the DR region. It is sequenced so the dependency order is preserved:
DNS → load balancer → database → workers → API.

### 3.1 DNS failover

```bash
# The apex domain (civic.example.org) is a Route 53 record set with a
# health-check on the primary ALB. When the health check fails twice
# (60 s), Route 53 automatically promotes the DR record set.

# To force failover immediately (skipping the health check):
aws route53 change-resource-record-sets \
  --hosted-zone-id Z<CIVIC_ZONE_ID> \
  --change-batch file://dns-failover.json

# dns-failover.json:
# {
#   "Changes": [{
#     "Action": "UPSERT",
#     "ResourceRecordSet": {
#       "Name": "civic.example.org.",
#       "Type": "A",
#       "AliasTarget": { "DNSName": "civic-dr-alb.us-east-2.elb.amazonaws.com", ... }
#     }
#   }]
# }
```

**TTL:** 60 seconds. The combination of TTL + Route 53 health checks
bounds the client-visible failover window to ~120 s.

### 3.2 Load balancer failover

The DR region's ALB is provisioned at deploy time and is always
"warm" — it just has zero targets until the DR region's API pods are
rolled. To failover:

```bash
# 1. Add the DR pods as targets of the DR ALB.
aws elbv2 register-targets \
  --target-group-arn arn:aws:elasticloadbalancing:...:dr-tg \
  --targets $(kubectl get pods -n civic-prod -l app=api -o jsonpath='{.items[*].status.podIP}' | tr ' ' ',')

# 2. Wait for the targets to become healthy.
aws elbv2 wait target-in-service --target-group-arn arn:...:dr-tg
```

### 3.3 Database failover

For Postgres, failover is **promote-then-cutover**: promote the DR
replica to primary, then point the API at the new primary.

```bash
# 1. Promote the DR replica to a standalone primary.
aws rds promote-read-replica --db-instance-identifier civic-pg-dr

# 2. Update the API's DATABASE_URL config map (see §2.2).
kubectl -n civic-prod edit configmap api-config
kubectl -n civic-prod rollout restart deployment/api

# 3. Verify the API can read from the new primary.
kubectl -n civic-prod exec deploy/api -- \
  curl -s http://localhost:9000/api/v1/healthz | jq .checks.db
# Expect: "up"
```

---

## 4. Failback procedure

Failback is the act of returning to the primary region after the DR
region has served traffic. It is the reverse of failover but with an
extra reconciliation step (§4.5) because the DR region may have
accepted writes the primary does not yet have.

### 4.1 Provision a fresh primary

```bash
# 1. Spin up a new Postgres in the primary region from the DR snapshot.
aws rds restore-db-instance-from-db-snapshot \
  --db-instance-identifier civic-pg-primary-new \
  --db-snapshot-identifier civic-pg-dr-$(date -u +%Y%m%d)

# 2. Set up logical replication FROM DR TO the new primary so the
#    writes accumulated during the DR window are streamed back.
psql -c "CREATE SUBSCRIPTION civic_failback CONNECTION 'host=civic-pg-dr...' PUBLICATION civic_all;"
```

### 4.2 Reconcile (catch the new primary up)

```bash
# Wait until the subscription's lag reaches zero.
psql -c "SELECT pg_wal_lsn_diff(pg_current_wal_lsn(), received_lsn) FROM pg_stat_subscription;"
# Expect: 0 bytes
```

### 4.3 Cut DNS back

```bash
aws route53 change-resource-record-sets \
  --hosted-zone-id Z<CIVIC_ZONE_ID> \
  --change-batch file://dns-failback.json
```

### 4.4 Tear down the DR region's replicas

```bash
aws rds delete-db-instance --db-instance-identifier civic-pg-dr --skip-final-snapshot
helm uninstall nats-dr
```

### 4.5 Verify (post-failback)

Run the verification checks in §5.

---

## 5. Data verification (post-restore checks)

Every restore ends with this checklist. **None of the steps are
optional** — a restore that fails verification is treated as a
continuing incident.

### 5.1 Row counts match the last good backup

```bash
# Run on the restored primary.
psql -c "SELECT
  (SELECT COUNT(*) FROM legislation.bills)        AS bills,
  (SELECT COUNT(*) FROM legislation.acts)         AS acts,
  (SELECT COUNT(*) FROM government.debt_snapshots) AS snapshots,
  (SELECT COUNT(*) FROM simulation.scenarios)     AS scenarios,
  (SELECT COUNT(*) FROM documents.documents)      AS documents,
  (SELECT COUNT(*) FROM identity.users)           AS users;"
```

Compare against the snapshot of these counts taken before the incident
(stored in `s3://civic-backups-sentinel/counts/`).

### 5.2 Latest timestamps are within the RPO

```bash
psql -c "SELECT
  (SELECT MAX(updated_at)     FROM legislation.bills)         AS latest_bill_update,
  (SELECT MAX(observation_date) FROM government.debt_snapshots) AS latest_snapshot,
  NOW() - (SELECT MAX(updated_at) FROM legislation.bills)     AS bill_lag;"
```

**Pass criterion:** `bill_lag` ≤ 1 hour (the RPO). If the lag exceeds
the RPO, file a post-incident report and adjust the backup cadence.

### 5.3 Object store inventory matches

```bash
aws s3 ls s3://civic-documents-primary --recursive --summarize \
  > /tmp/post-restore-inventory.txt
diff /tmp/pre-incident-inventory.txt /tmp/post-restore-inventory.txt
```

### 5.4 NATS JetStream consumers are alive

```bash
nats stream info CIVIC_EVENTS
nats stream info CIVIC_BILL_PROCESSING
nats consumer ls CIVIC_EVENTS
```

Every consumer's `last_active` should be within the last 60 seconds
(workers should be running and consuming again).

### 5.5 API healthz returns 200 with all checks up

```bash
curl -s http://localhost:8080/api/v1/healthz | jq .
# Expected:
# {
#   "status": "healthy",
#   "checks": { "db": "up", "redis": "up", "nats": "up", "s3": "up" }
# }
```

### 5.6 Spec §75 critical-failure test (worker crash + resume)

The critical failure test (`tests/chaos/critical_failure_test.go`)
simulates a worker crash mid-Bill-Processing-Pipeline. After a restore:

1. Trigger a fresh Bill Processing workflow.
2. Kill the worker pod mid-flight (between Fetch and Parse).
3. Spin up a new worker.
4. Verify the workflow RESUMES from the last completed activity (Fetch
   is NOT re-executed — content hash is idempotent on RunID).
5. Verify the workflow completes with `validated=true` (or
   `validated=false` if evidence was insufficient — either is OK; a
   FAILED state is NOT).

The test must pass before the restore is declared complete.

---

## 6. Cross-references

- `docs/PRODUCTION_GATE.md` §2 row 12 — the gate this document closes.
- `tests/chaos/README.md` — chaos runbooks (Gate #11) that assume the
  backup + restore steps above as the recovery path.
- `tests/chaos/critical_failure_test.go` — spec §75 critical failure
  test, run after every restore (§5.6).
- `infrastructure/terraform/modules/rds/main.tf` — RDS configuration
  (multi-AZ, automated backups, WAL archive).
- `infrastructure/terraform/modules/s3/main.tf` — S3 cross-region
  replication config.
- `infrastructure/terraform/modules/nats/main.tf` — NATS JetStream
  snapshot schedule.
- `infrastructure/kubernetes/helm/civic-intelligence/values-prod.yaml`
  — production values (DB URL, bucket name, NATS URL) updated during
  failover.
- `ADR-0007-temporal-for-durable-workflows.md` — why Temporal survives
  worker crashes (the foundation for the §75 critical-failure test).
- `ADR-0003-postgres-single-cluster-logical-schemas.md` — why we use a
  single Postgres cluster (simplifies backup + restore).

---

## 7. Cadence

| Drill                                | Frequency                          | Owner           |
|--------------------------------------|------------------------------------|-----------------|
| Backup verification (sentinel check)  | Continuous                         | SRE on-call     |
| Single-table recovery drill           | Monthly (first Monday)              | SRE on-call     |
| Full DR failover + failback           | Quarterly (game-day)                | Platform team    |
| §75 critical-failure test             | Every PR touching infra code        | CI (automated)   |
| RPO / RTO audit                       | Annually                            | SRE lead         |

When a drill fails, follow `tests/chaos/README.md` §"When a drill
fails" — capture the failure mode, file an issue, and tag the runbook
with `Status: FAILING — see #<issue>` until fixed.
