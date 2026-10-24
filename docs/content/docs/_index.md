---
title: "Documentation"
weight: 1
bookFlatSection: true
---

# eds-inventory Documentation

This site is both the **developer guide** and the **user guide** for
eds-inventory. It grows alongside the services rather than after them, as
EdS's does.

- **Requirements** says what the inventory must do, and what is still open.
- **Architecture** says *why* it is shaped the way it is, and which part is
  permitted to make which decision.
- **User Guide** is for operating what exists: adding a source, running the
  services, keeping the inventory in step.
- **Developer Guide** is for changing what exists: building, testing, and the
  steps that are easy to get wrong once and painful to rediscover.
- **Reference** is the contract. It is what clients depend on, so it is
  described precisely and versioned explicitly.

Those are **maintained**: kept current as the system changes. Three kinds of
record are **retained** instead, written once and left as evidence:

- **[Decisions](/docs/architecture/decisions/)** (ADRs) say why a design fork
  went the way it did, and what was turned down.
- **[Change Records](/docs/changes/)** say what was changed on something
  running, and what actually happened.
- **[Incident Records](/docs/incidents/)** say what broke, why, and what was
  done about it.

[Keeping Records](/docs/develop/records/) has the rules and the templates.
