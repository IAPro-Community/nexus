import { describe, expect, it } from 'vitest';
import { buildDiffCommentPrompt } from './ProjectDiffDialog';

describe('ProjectDiffDialog helpers', () => {
  it('formats a structured comment-to-agent prompt with branch, workspace, and file focus', () => {
    const prompt = buildDiffCommentPrompt(
      { branch: 'release/v1.0.0', dir: '/projetos/nexus' },
      'Ajustar validação de worktree no endpoint',
      'internal/control/web/handlers_git.go',
    );
    expect(prompt).toContain('[Git Diff Review · release/v1.0.0]');
    expect(prompt).toContain('Workspace: /projetos/nexus');
    expect(prompt).toContain('File: internal/control/web/handlers_git.go');
    expect(prompt).toContain('Ajustar validação de worktree no endpoint');
  });
});
