import { describe, expect, it } from 'vitest';
import { defaultMissionAutonomyContract } from './missionAutonomyModel';

describe('mission autonomy defaults', () => {
  it('defaults to local-first verification with dangerous capabilities denied', () => {
    const contract = defaultMissionAutonomyContract();
    expect(contract.require_verification).toBe(true);
    expect(contract.auto_remediate).toBe(true);
    expect(contract.disallow_destructive_git).toBe(true);
    expect(contract.allow_git_push).toBe(false);
    expect(contract.allow_deploy).toBe(false);
    expect(contract.allow_external_network).toBe(false);
    expect(contract.allow_secret_access).toBe(false);
    expect(contract.allow_paid_services).toBe(false);
    expect(contract.allow_computer_use).toBe(false);
    expect(contract.allow_screen_capture).toBe(false);
    expect(contract.allow_keyboard_input).toBe(false);
    expect(contract.allow_pointer_input).toBe(false);
    expect(contract.allow_window_management).toBe(false);
    expect(contract.max_retries).toBeGreaterThan(0);
    expect(contract.max_total_iterations).toBeGreaterThan(contract.max_retries);
    expect(contract.max_no_progress).toBeGreaterThan(0);
  });
});
