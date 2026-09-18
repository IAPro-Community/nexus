# Legacy quota cache migration

Snapshots without a verifiable `AccountScope` are legacy observations. The
scoped cache API quarantines them logically by returning `UNKNOWN`; it does not
delete the original `usage.json` or `quota.json`. Automatic migration is only
safe after an exact provider, profile and authenticated identity match, followed
by a fresh scoped observation.
