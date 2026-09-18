# Account isolation

Account-owned state is scoped by `model.AccountScope`, whose canonical key is
`provider_id/account_id/identity_version`. Provider and profile names and email
addresses are labels only. New quota persistence must use
`SaveUsageForScope`; missing or mismatched scope is returned as `UNKNOWN` and
is never attributed to the active account.

Cooldowns and quota-monitor notification state use the same scope when supplied.
Legacy constructors remain temporarily for compatibility and are explicitly
stored under a `legacy/` boundary.
