---
title: "Change Records"
weight: 50
bookCollapseSection: true
---

# Change Records

One record per significant change to something that is **running**: the
tenant, a workload, the database. What was changed and why, the end
state it aimed for, the procedure, how to undo it, and what actually happened.

These are different from the rest of the site. The guides and reference pages
are **maintained**: kept current, edited whenever reality moves. A change
record is **retained**: evidence of what was done on a given day, written once
and then left alone apart from its follow-ups closing.
[Incidents](/docs/incidents/) are kept the same way.

## When a change needs one

Write a record when the change touches something that is already running and
would be hard to reconstruct from the commit alone:

- a `terraform apply` against the tenant that adds, replaces or removes
  something issued
- a deployment to the workload, or a new service on it
- a migration of the database, or a source added or removed
- a credential rotated, an account replaced, an authorization renewed

A change confined to the repository — code, tests, documentation — does not
need one; the commit is its record. A design choice is not a change: that is
an [ADR](/docs/architecture/decisions/).

New records start from the
[change record template](/docs/develop/records/change-record/). Records are
numbered `CHG-NNNN` in the order they are opened, across years, and a number
is never reused, even for a change that was abandoned.

A bare **CHG-0001** means an eds-inventory record. Anyone else's is written
with its owner: **EdS CHG-0040**, **Deevnet CHG-0021**.

## Records

| ID | Date | Change | Type | Status |
|---|---|---|---|---|
| CHG-0001 | 2026-10-24 | [inventoryd on EdS's Workload](2026/0001-inventoryd-on-eds-workload/) | Deployment | {{< status-badge "planned" "Planned" >}} |
