# Provider registration lifecycle

Provider binary discovery and account registration are separate concerns.
Current provider detection reports installation (`installed`, version and path),
while profile registration owns `account-scope.json` and authentication state.
This boundary permits an installed CLI to remain unregistered without creating
quota or status data for an account. The interactive registration states and
ephemeral execution policy remain the next lifecycle slice.
