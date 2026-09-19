#!/usr/bin/env bash
# Apply branch protection to main (requires a GitHub admin token).
# Usage: GITHUB_TOKEN=<admin-token> ./scripts/apply-branch-protection.sh
#
# Required checks must match job names in .github/workflows/ci.yml for the
# certified SHA. Review and adjust REQUIRED_CHECKS when CI job names change.
# This script is the EXTERNAL_GITHUB_ADMIN_GATE executable form: it is not
# done until it runs successfully with an admin token.
set -euo pipefail

REPO="${GITHUB_REPO:-IAPro-Community/nexus}"
BRANCH="${PROTECT_BRANCH:-main}"
: "${GITHUB_TOKEN:?Set GITHUB_TOKEN to a token with admin:repo scope}"

# TODO(owner): pin the exact required check names to ci.yml job IDs.
read -r -d '' CHECKS_JSON <<'EOF' || true
[{"context": "Frontend (Format, Lint, Styles, Typecheck, Unit & Build)"}, {"context": "Security"}, {"context": "Browser E2E, Axe & visual smoke"}]
EOF

curl -fsSL -X PUT \
  -H "Authorization: Bearer ${GITHUB_TOKEN}" \
  -H "Accept: application/vnd.github+json" \
  "https://api.github.com/repos/${REPO}/branches/${BRANCH}/protection" \
  -d "$(cat <<EOF
{
  "required_status_checks": {"strict": true, "checks": ${CHECKS_JSON}},
  "enforce_admins": true,
  "required_pull_request_reviews": {"required_approving_review_count": 1, "dismiss_stale_reviews": true},
  "restrictions": null,
  "allow_force_pushes": false,
  "allow_deletions": false
}
EOF
)"

echo "Protection applied to ${REPO}@${BRANCH}."
