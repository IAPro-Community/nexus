import { describe, expect, it } from 'vitest';
import {
  buildProjectRoute,
  buildPopoutRoute,
  parseRouteLocation,
  routeToWorkspaceSurface,
  globalSurfaceToProjectSurface,
} from './routes';
import { projectSurface } from './surfaces';

describe('Semantic Route Parsing and Building', () => {
  it('parses root and global routes', () => {
    expect(parseRouteLocation('/')).toEqual({ kind: 'root' });
    expect(parseRouteLocation('/projects')).toEqual({ kind: 'global', surface: 'projects' });
    expect(parseRouteLocation('/settings')).toEqual({ kind: 'global', surface: 'settings' });
    expect(parseRouteLocation('/updates')).toEqual({ kind: 'global', surface: 'updates' });
    expect(parseRouteLocation('/welcome')).toEqual({ kind: 'global', surface: 'welcome' });
  });

  it('parses project-scoped overview route default', () => {
    expect(parseRouteLocation('/p/proj-123')).toEqual({
      kind: 'project',
      projectId: 'proj-123',
      surface: 'overview',
    });
  });

  it('parses project surfaces', () => {
    expect(parseRouteLocation('/p/proj-123/missions')).toEqual({
      kind: 'project',
      projectId: 'proj-123',
      surface: 'missions',
    });
    expect(parseRouteLocation('/p/proj-123/work')).toEqual({
      kind: 'project',
      projectId: 'proj-123',
      surface: 'work',
    });
    expect(parseRouteLocation('/p/proj-123/terminals')).toEqual({
      kind: 'project',
      projectId: 'proj-123',
      surface: 'terminals',
    });
    expect(parseRouteLocation('/p/proj-123/agents')).toEqual({
      kind: 'project',
      projectId: 'proj-123',
      surface: 'agents',
    });
    expect(parseRouteLocation('/p/proj-123/resources')).toEqual({
      kind: 'project',
      projectId: 'proj-123',
      surface: 'resources',
    });
  });

  it('parses deep-link routes for flow-run, terminals, and agents', () => {
    const missionRoute = parseRouteLocation('/p/proj-123/missions/run-456');
    expect(missionRoute).toEqual({
      kind: 'project',
      projectId: 'proj-123',
      surface: 'missions',
      subId: 'run-456',
    });
    expect(routeToWorkspaceSurface(missionRoute)).toMatchObject({
      type: 'flow-run',
      data: { runId: 'run-456' },
    });

    expect(parseRouteLocation('/p/proj-123/work/session-789')).toEqual({
      kind: 'project',
      projectId: 'proj-123',
      surface: 'work',
      subId: 'session-789',
    });

    expect(parseRouteLocation('/p/proj-123/terminals/rt-999')).toEqual({
      kind: 'project',
      projectId: 'proj-123',
      surface: 'terminals',
      subId: 'rt-999',
    });

    expect(parseRouteLocation('/p/proj-123/agents/agent-1')).toEqual({
      kind: 'project',
      projectId: 'proj-123',
      surface: 'agents',
      subId: 'agent-1',
    });

    expect(parseRouteLocation('/p/proj-123/agents/agent-1/config')).toEqual({
      kind: 'project',
      projectId: 'proj-123',
      surface: 'agents',
      subId: 'agent-1',
      action: 'config',
    });
  });

  it('parses popout route isolation', () => {
    expect(parseRouteLocation('/p/proj-123/popout/terminals')).toEqual({
      kind: 'popout',
      projectId: 'proj-123',
      surface: 'terminals',
    });
    expect(parseRouteLocation('/p/proj-123/popout/work')).toEqual({
      kind: 'popout',
      projectId: 'proj-123',
      surface: 'work',
    });
  });

  it('returns root for malformed percent-encoding instead of throwing', () => {
    expect(() => parseRouteLocation('/p/%E0%A4%A/settings')).not.toThrow();
    expect(parseRouteLocation('/p/%E0%A4%A/settings')).toEqual({ kind: 'root' });
  });

  it('maps global update route to the settings workspace surface', () => {
    expect(globalSurfaceToProjectSurface('updates')).toBe('settings');
    expect(globalSurfaceToProjectSurface('settings')).toBe('settings');
    expect(globalSurfaceToProjectSurface('welcome')).toBeNull();
  });

  it('builds canonical project and popout routes', () => {
    expect(buildProjectRoute('proj-1')).toBe('/p/proj-1/overview');
    expect(buildProjectRoute('proj-1', 'missions')).toBe('/p/proj-1/missions');
    expect(buildProjectRoute('proj-1', 'missions', 'run-456')).toBe('/p/proj-1/missions/run-456');
    expect(buildProjectRoute('proj-1', 'agents', 'agent-1', 'config')).toBe(
      '/p/proj-1/agents/agent-1/config',
    );
    expect(buildPopoutRoute('proj-1', 'terminals')).toBe('/p/proj-1/popout/terminals');
  });

  it('maps parsed routes to matching WorkspaceSurfaces', () => {
    const route1 = parseRouteLocation('/p/proj-123/overview');
    const surf1 = routeToWorkspaceSurface(route1);
    expect(surf1).toEqual(projectSurface('proj-123', 'overview'));

    const route2 = parseRouteLocation('/p/proj-123/missions/run-456');
    const surf2 = routeToWorkspaceSurface(route2);
    expect(surf2?.type).toBe('flow-run');
    expect(surf2?.data?.runId).toBe('run-456');

    const route3 = parseRouteLocation('/p/proj-123/agents/agent-1');
    const surf3 = routeToWorkspaceSurface(route3, {
      agents: [
        { id: 'agent-1', name: 'Claude Code', role: 'architect', provider: 'claude' } as any,
      ],
    });
    expect(surf3?.type).toBe('terminal');
    expect(surf3?.data?.agentId).toBe('agent-1');
    expect(surf3?.title).toBe('Claude Code');

    const route4 = parseRouteLocation('/p/proj-123/agents/agent-1/config');
    const surf4 = routeToWorkspaceSurface(route4, {
      agents: [
        { id: 'agent-1', name: 'Claude Code', role: 'architect', provider: 'claude' } as any,
      ],
    });
    expect(surf4?.type).toBe('agent-config');
    expect(surf4?.data?.agentId).toBe('agent-1');

    const route5 = parseRouteLocation('/p/proj-123/terminals/rt-999');
    const surf5 = routeToWorkspaceSurface(route5);
    expect(surf5?.type).toBe('project-shell');
    expect(surf5?.data?.runtimeId).toBe('rt-999');
  });
});
