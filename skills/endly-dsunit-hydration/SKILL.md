---
name: endly-dsunit-hydration
description: Hydrate Endly use-case fixtures over preloaded dictionaries, with database mappings, sequence-backed IDs, and published per-case state.
---

Keep three layers distinct: schema/connections, stable dictionary data, and mutable
case data. Initialize/register the datastore and load its dictionary before the
regression batch. Reset only the mutable tables needed by the batch or case;
repopulate dictionary rows if the reset also affects them. Preloaded data is part
of a focused case's prerequisites even when the infrastructure task is omitted.

Template `data` references can aggregate case fixtures into workflow data before
setup runs. Preserve each case's tag/Key so hydration publishes rows back to the
correct case. For table-keyed fixtures, query dsunit:sequence before AsTableRecords;
use symbolic sequence keys and a namespace suffix to publish records by case.
For TableData fixtures, preserve Table, Data, Key, AutoGenerate and PostIncrement.
Their output can be read from `dsunit.<case-key>` in a case's init.

When a logical fixture spans physical tables: register the connection, load
`dsunit:mapping`, obtain its Tables, query sequence values for those physical
tables, publish the sequence namespace used by the fixture, convert the fixture
once with AsTableRecords, then dsunit:prepare. Mapping precedes sequence discovery
and population. Do not replace logical dataset names with physical table names
or discard associations/default values to work around a failed setup.

Use case init to hydrate the published rows, load case-specific request/expectation
resources, and derive IDs/foreign keys. Read defaults through the ordinary Endly
instance -> workflow/default -> workflow fallback. A missing optional fixture is
different from a missing required resource. A fresh rerun must use original fixture
expressions; AsTableRecords mutates source records and some forms are cached.
Reload/rebuild fixture data when new allocation is needed rather than converting
already expanded numeric IDs again.

If the application uses a snapshot or derived cache, populate fixtures before its
index/refresh step and start/readiness check. Database presence alone does not prove
runtime hydration. For a focused case, retain that dependency order and inspect
both the published fixture state and observed API/database results.

Retrieve endly-dsunit-mapping, endly-dsunit-sequences and endly-dsunit-prepare for
the relevant contracts via endly_service_info; use endly-testing for methodology.
