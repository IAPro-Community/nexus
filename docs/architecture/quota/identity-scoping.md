# Identity scoping

Registered profiles persist an opaque `account_id` in `account-scope.json`.
When the authenticated identity changes, `identity_version` increments while
the account record remains stable. An empty identity cannot upgrade a snapshot
to attributable data. The identity value is local metadata and is not emitted
to logs.
