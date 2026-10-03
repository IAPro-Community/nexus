import type { AutonomyContract } from '../../types';

// Mirrors runner.DefaultAutonomyContract. The web UI always sends the complete
// approved contract so the durable Mission snapshot records exactly what the
// user authorized.
export function defaultMissionAutonomyContract(): AutonomyContract {
  return {
    max_retries: 3,
    max_total_iterations: 120,
    max_no_progress: 2,
    package_timeout_seconds: 3600,
    auto_remediate: true,
    require_verification: true,
    disallow_destructive_git: true,
    allowed_file_patterns: [],
    verification_commands: [],
    escalate_on_failure: true,
    allow_tool_auto_approval: true,
    allow_git_push: false,
    allow_deploy: false,
    allow_external_network: false,
    allow_secret_access: false,
    allow_paid_services: false,
    allow_computer_use: false,
    allow_screen_capture: false,
    allow_keyboard_input: false,
    allow_pointer_input: false,
    allow_window_management: false,
  };
}

export function linesToList(value: string): string[] {
  return value
    .split(/\r?\n/)
    .map((item) => item.trim())
    .filter(Boolean);
}

export function listToLines(value?: string[]): string {
  return (value || []).join('\n');
}
