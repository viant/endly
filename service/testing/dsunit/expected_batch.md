# Deferred group expectations

`dsunit:collectExpected` loads a case's expected datasets and resolves variables
against that case's current state. It appends the frozen rows to
`<Bucket>.<Datastore>` without querying or changing the database. Case hydration
must happen first. Collect only dedicated case-owned expectations; unrelated
cases must not mutate the same fixture identities.

```yaml
collect:
  action: dsunit:collectExpected
  bucket: groupExpected.delivery
  datastore: ci_ads
  URL: ${path}/expect
```

After all group cases have executed, use the existing DSUnit expectation action:

```yaml
verifyGroup:
  action: dsunit:expect
  datastore: ci_ads
  checkPolicy: 1
  data: $groupExpected.delivery.ci_ads
```

Collection is not verification. The caller must retain UI failures, require the
final database batch to pass, and avoid marking cases successful merely because
expected rows were collected. Each datastore has its own batch. Reset the bucket
before a new group run. Existing expected-row directives are retained.
