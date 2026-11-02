#!/usr/bin/env python3
"""Deterministic Chromium E2E for the current Nexus web bundle.

The execution environment used by ChatGPT blocks all browser navigation by
administrator policy (including loopback and file://). This harness therefore:
- opens a real Chromium page through Playwright;
- installs browser-native deterministic localStorage/fetch/WebSocket fixtures;
- injects the freshly built bundle.js and bundle.css directly;
- exercises the actual React/XTerm application and asserts the canonical UX.

The fixture's fetch implementation requires the CSRF token obtained from the
session bootstrap response on every mutation. Backend cookie/Origin semantics
remain covered by internal/control/web auth/E2E tests and are a separate gate.
"""
from __future__ import annotations

import argparse
import json
import traceback
from pathlib import Path
from typing import Any

from playwright.sync_api import expect, sync_playwright

ROOT = Path(__file__).resolve().parents[1]
DIST = ROOT / "web" / "dist"
NOW = "2026-09-01T10:30:00Z"
CSRF = "nexus-e2e-csrf"

PROJECT = {
    "id": "prj-e2e", "name": "Nexus E2E", "slug": "nexus-e2e",
    "canonical_path": "/tmp/nexus-e2e", "repo_remote": "origin",
    "repo_url": "https://example.invalid/nexus-e2e.git", "default_branch": "main",
    "maestro_mode": "ASSIST", "resource_policy": "BALANCED",
    "default_isolation": "NONE", "settings": "{}", "created_at": NOW, "updated_at": NOW,
}
AGENT = {
    "id": "agt-existing", "project_id": PROJECT["id"], "name": "Existing Agent",
    "role": "engineer", "status": "WORKING", "continuity_status": "BOUND",
    "created_at": NOW, "updated_at": NOW,
}

def pkg(pid: str, title: str, deps: list[str], strategy: str, role: str, agent: str = "") -> dict[str, Any]:
    return {
        "id": pid, "title": title, "goal": f"Execute {title}", "priority": "NORMAL", "status": "READY",
        "dependencies": deps, "parallel_group": "parallel-bc" if pid in {"B", "C"} else "",
        "role": role, "agent_allocation": agent, "assignment_strategy": strategy,
        "resource_policy": "BALANCED", "provider": "fake" if agent else "", "profile": "default" if agent else "",
        "maestro_gates": [], "maestro_skills": ["verification"] if pid == "D" else [],
        "relevant_paths": ["internal"] if pid in {"A", "B"} else ["web/src"],
        "acceptance_criteria": [f"{title} is complete and verified"],
        "verification_requirements": ["echo NEXUS_VERIFY_OK"], "shared_artifacts": [],
    }

PLAN = {
    "id": "plan-e2e", "project_id": PROJECT["id"], "title": "Canonical A to B/C to D Flow",
    "description": "Deterministic E2E Flow", "status": "DRAFT", "current_revision": 1,
    "structured_facts": {"nexus.flow_policy": "GUIDED"}, "created_at": NOW, "updated_at": NOW,
    "phases": [
        {"id": "phase-build", "title": "Build", "description": "Build in waves", "order": 1, "packages": [
            pkg("A", "A · Architecture", [], "EXISTING", "architect", "agt-existing"),
            pkg("B", "B · Backend", ["A"], "CREATE", "backend"),
            pkg("C", "C · Frontend", ["A"], "AUTO", "frontend"),
        ]},
        {"id": "phase-verify", "title": "Verify", "description": "Join and verify", "order": 2,
         "packages": [pkg("D", "D · Verification", ["B", "C"], "AUTO", "tester")]},
    ],
}

def receipt(step: str, deps: list[str]) -> dict[str, Any]:
    return {
        "id": f"receipt-{step}", "run_id": "run-e2e", "step_id": step, "status": "VERIFIED",
        "summary": f"{step} completed with deterministic evidence", "changed_files": [f"e2e/{step.lower()}.txt"],
        "commands": ["echo NEXUS_VERIFY_OK"], "tests": [], "verification": [], "decisions": [], "artifacts": [],
        "remaining_issues": [], "agent_id": "agt-existing" if step == "A" else f"agt-{step.lower()}",
        "base_revision": "base0001", "result_revision": f"result-{step.lower()}", "started_at": NOW, "completed_at": NOW,
        "dependency_ids": deps,
    }

RECEIPTS = {s: receipt(s, deps) for s, deps in {"A": [], "B": ["A"], "C": ["A"], "D": ["B", "C"]}.items()}
CAPSULES = [{
    "id": f"capsule-{step}", "run_id": "run-e2e", "project_id": PROJECT["id"], "branch": "main",
    "head": "abcdef0123456789", "dirty": False, "context_fingerprint": "fp-e2e", "flow_id": PLAN["id"],
    "flow_revision": 2, "step": {"id": step, "title": f"Step {step}", "goal": f"Execute {step}", "dependencies": deps,
    "role": "engineer", "assignment_strategy": "AUTO", "acceptance_criteria": [f"{step} verified"],
    "verification_requirements": ["echo NEXUS_VERIFY_OK"]},
    "dependency_receipts": [RECEIPTS[d] for d in deps], "relevant_paths": ["internal", "web/src"],
    "maestro_skills": [], "acceptance_criteria": [f"{step} verified"], "constraints": ["Stay inside approved Flow Step"],
    "created_at": NOW,
} for step, deps in {"A": [], "B": ["A"], "C": ["A"], "D": ["B", "C"]}.items()]

def run_fixture() -> dict[str, Any]:
    items = PLAN["phases"][0]["packages"] + PLAN["phases"][1]["packages"]
    return {
        "id": "run-e2e", "plan_id": PLAN["id"], "project_id": PROJECT["id"], "plan_revision": 2,
        "execution_snapshot_id": "snapshot-e2e", "state": "COMPLETED_VERIFIED", "paused_reason": "",
        "current_pkg_index": 3, "total_iterations": 4,
        "contract": {"max_retries": 2, "max_total_iterations": 12, "require_verification": True,
                     "verification_commands": ["echo NEXUS_VERIFY_OK"]},
        "package_runs": [{
            "id": f"pkgrun-{item['id']}", "package_id": item["id"], "title": item["title"], "goal": item["goal"],
            "state": "VERIFIED", "attempt": 1, "dependencies": item["dependencies"],
            "assigned_agent": "agt-existing" if item["id"] == "A" else f"agt-{item['id'].lower()}",
            "assignment_strategy": item["assignment_strategy"], "provider": "fake", "profile": "default",
            "capsule_id": f"capsule-{item['id']}", "receipt_id": f"receipt-{item['id']}",
            "dispatch_id": f"dispatch-{item['id']}", "error_message": "", "started_at": NOW,
        } for item in items],
        "started_at": NOW, "updated_at": NOW,
    }

FIXTURES = json.dumps({"project": PROJECT, "agent": AGENT, "plan": PLAN, "run": run_fixture(),
                       "capsules": CAPSULES, "receipts": list(RECEIPTS.values()), "csrf": CSRF})

BROWSER_FIXTURE = r"""
(cfg => {
  const memory = new Map();
  const storage = { getItem:k=>memory.has(k)?memory.get(k):null, setItem:(k,v)=>memory.set(String(k),String(v)),
    removeItem:k=>memory.delete(k), clear:()=>memory.clear(), key:i=>[...memory.keys()][i]??null, get length(){return memory.size;} };
  Object.defineProperty(window,'localStorage',{configurable:true,value:storage});
  Object.defineProperty(window,'sessionStorage',{configurable:true,value:storage});

  const state = window.__NEXUS_E2E_STATE__ = { contextReady:false, plan:null, revisions:[], askCalls:[], shellStarts:0,
    stoppedRuntimes:[], writes:[], csrfFailures:0, fetches:[] };
  const J = (payload,status=200) => Promise.resolve(new Response(JSON.stringify(payload),{status,headers:{'Content-Type':'application/json'}}));
  const body = opts => { try { return opts?.body ? JSON.parse(opts.body) : {}; } catch { return {}; } };
  const headers = opts => new Headers(opts?.headers || {});
  const mutation = (path,opts) => {
    if(headers(opts).get('X-CSRF-Token') !== cfg.csrf){ state.csrfFailures++; return false; }
    state.writes.push([String(opts?.method||'POST').toUpperCase(),path]); return true;
  };
  window.fetch = async (input, opts={}) => {
    const raw = typeof input === 'string' ? input : input.url; const u = new URL(raw,'http://nexus.e2e'); const path=u.pathname;
    const method=String(opts.method||'GET').toUpperCase(); state.fetches.push([method,path]);
    if(method!=='GET' && method!=='HEAD' && !mutation(path,opts)) return J({error:'invalid CSRF token'},403);
    if(path==='/api/v1/session') return J({authenticated:true,csrf_token:cfg.csrf});
    if(path==='/api/v1/projects') return J([cfg.project]);
    if(path===`/api/v1/projects/${cfg.project.id}`) return J({project:cfg.project,layout:''});
    if(path===`/api/v1/projects/${cfg.project.id}/agents`) return J([cfg.agent]);
    if(path===`/api/v1/projects/${cfg.project.id}/context`) return J({project_id:cfg.project.id,state:state.contextReady?'READY':'MISSING',
      current_fingerprint:{project_id:cfg.project.id,canonical_path:cfg.project.canonical_path,branch:'main',head:'abcdef0123456789',dirty_fingerprint:'',maestro_version:'1.0.0'},
      current_fingerprint_id:'fp-e2e',hydrated_fingerprint_id:state.contextReady?'fp-e2e':'',maestro_available:true,maestro_version:'1.0.0',updated_at:'2026-09-01T10:30:00Z'});
    if(path===`/api/v1/projects/${cfg.project.id}/context/prepare`){state.contextReady=true;return J({project_id:cfg.project.id,state:'READY',current_fingerprint:{project_id:cfg.project.id,canonical_path:cfg.project.canonical_path,branch:'main',head:'abcdef0123456789',dirty_fingerprint:'',maestro_version:'1.0.0'},current_fingerprint_id:'fp-e2e',hydrated_fingerprint_id:'fp-e2e',maestro_available:true,maestro_version:'1.0.0',updated_at:'2026-09-01T10:30:00Z'});}
    if(path==='/api/v1/workspaces') return J([{id:'ws-e2e',name:'Nexus E2E',path:cfg.project.canonical_path,is_active:true}]);
    if(path==='/api/v1/runtimes') return J([{runtime_id:'rt-agent',agent_id:cfg.agent.id,title:'Existing Agent',provider_id:'fake',profile_id:'default',workspace:cfg.project.canonical_path,pid:101,host_pid:101,state:'RUNNING',control_level:'TERMINAL',control_endpoint:'mock',started_at:'2026-09-01T10:30:00Z'}]);
    if(path==='/api/v1/providers') return J([{id:'fake',installed:true,version:'1.0-e2e',control_level:'TERMINAL',capabilities:{}}]);
    if(path==='/api/v1/profiles') return J([{name:'default',provider:'fake',authenticated:true,is_default:true}]);
    if(path==='/api/v1/events') return J([]);
    if(path==='/api/v1/system/updates') return J({nexus_version:'e2e',maestro_version:'1.0.0',maestro_available:true,update_available:false});
    if(path===`/api/v1/agents/${cfg.agent.id}/ask`){state.askCalls.push(body(opts));return J({agent_id:cfg.agent.id,runtime_id:'rt-agent',started:false,accepted:true});}
    if(path===`/api/v1/agents/${cfg.agent.id}`) return J({agent:cfg.agent,generations:[],lineage:[],revisions:[]});
    if(path===`/api/v1/projects/${cfg.project.id}/shell`){state.shellStarts++;return J({runtime:{runtime_id:'rt-shell-1',title:'Project Shell',provider_id:'shell',profile_id:'local',workspace:cfg.project.canonical_path,pid:202,host_pid:202,state:'RUNNING',control_level:'TERMINAL',control_endpoint:'mock',started_at:'2026-09-01T10:30:00Z'}},201);}
    if(path.startsWith('/api/v1/runtimes/')&&path.endsWith('/stop')){state.stoppedRuntimes.push(path.split('/')[4]);return J({status:'stopped'});}
    if(path.startsWith('/api/v1/runtimes/')&&path.endsWith('/title')) return J({status:'ok',title:body(opts).title||''});
    if(path===`/api/v1/projects/${cfg.project.id}/layout`) return J({status:'saved'});
    if(path===`/api/v1/projects/${cfg.project.id}/plans`){ if(method==='POST'){state.plan=structuredClone(cfg.plan);return J(state.plan,201);} return J(state.plan?[state.plan]:[]); }
    if(path==='/api/v1/schedules') return J([]);
    if(path===`/api/v1/plans/${cfg.plan.id}`){ if(method==='PUT'){const b=body(opts);state.plan=structuredClone(b.plan);state.plan.current_revision=Math.max(2,(state.plan.current_revision||1)+1);const rev={id:'rev-e2e',plan_id:cfg.plan.id,revision:state.plan.current_revision,snapshot_json:JSON.stringify(state.plan),change_summary:b.change_summary||'E2E',created_at:'2026-09-01T10:30:00Z'};state.revisions.unshift(rev);return J({plan:state.plan,revision:rev});} return J({plan:state.plan||cfg.plan,revisions:state.revisions}); }
    if(path===`/api/v1/plans/${cfg.plan.id}/run`) return J(cfg.run,201);
    if(path==='/api/v1/runs') return J([cfg.run]);
    if(path===`/api/v1/runs/${cfg.run.id}`) return J(cfg.run);
    if(path===`/api/v1/runs/${cfg.run.id}/evidence`) return J({run_id:cfg.run.id,capsules:cfg.capsules,receipts:cfg.receipts});
    if(path.startsWith(`/api/v1/runs/${cfg.run.id}/`)) return J(cfg.run);
    if(path==='/api/v1/maestro') return J({available:true,mode:'ASSIST',capabilities:{version:'1.0.0',skills:['verification']}});
    return J({});
  };

  class MockWS {
    static CONNECTING=0; static OPEN=1; static CLOSING=2; static CLOSED=3;
    constructor(url){this.url=String(url);this.readyState=0;this.sent=[];setTimeout(()=>{this.readyState=1;this.onopen?.({});this.onmessage?.({data:JSON.stringify({type:'lease',role:'CONTROL'})});const d=this.url.includes('rt-shell-1')?'NEXUS_SHELL_OK\\r\\n':'NEXUS_AGENT_OK\\r\\n';this.onmessage?.({data:JSON.stringify({type:'output',data:d})});},25);}
    send(v){this.sent.push(v);} close(){this.readyState=3;} addEventListener(){} removeEventListener(){}
  }
  window.WebSocket=MockWS;
})(__CFG__);
""".replace('__CFG__', FIXTURES)


def run_e2e(artifacts: Path) -> dict[str, Any]:
    css=(DIST/'bundle.css').read_text(); js=(DIST/'bundle.js').read_text()
    artifacts.mkdir(parents=True,exist_ok=True)
    errors=[]
    with sync_playwright() as p:
        browser=p.chromium.launch(headless=True,executable_path='/usr/bin/chromium',args=['--no-sandbox','--disable-dev-shm-usage'])
        page=browser.new_page(viewport={"width":1600,"height":1000})
        page.on('pageerror',lambda e:errors.append(f'pageerror: {e}'))
        page.on('console',lambda m:errors.append(f'console.error: {m.text}') if m.type=='error' else None)
        try:
            page.set_content(f'<html><head><style>{css}</style></head><body><div id="root"></div></body></html>')
            page.evaluate(BROWSER_FIXTURE)
            page.add_script_tag(content=js)
            expect(page.get_by_text('Nexus E2E',exact=True).first).to_be_visible(timeout=15000)

            page.keyboard.press('Control+K'); page.get_by_text('Open Composer',exact=True).click()
            expect(page.get_by_text('COMPOSER',exact=True)).to_be_visible(); expect(page.get_by_text('Context MISSING',exact=True)).to_be_visible()
            composer=page.locator('.nx-composer-surface'); composer.locator('textarea').first.fill('Implement the canonical deterministic Flow')
            expect(composer.get_by_role('button',name='Turn into Flow')).to_be_disabled()

            composer.get_by_role('button',name='Send to Agent').click()
            expect(page.get_by_text('Existing Agent', exact=True).first).to_be_visible(timeout=5000)
            state=page.evaluate('window.__NEXUS_E2E_STATE__'); assert len(state['askCalls'])==1
            close_agent=page.get_by_label('Close Existing Agent');
            if close_agent.count(): close_agent.first.click()

            page.keyboard.press('Control+K'); page.get_by_text('New Project Shell',exact=True).click()
            expect(page.get_by_text('Project Shell',exact=True).first).to_be_visible(timeout=5000)
            expect(page.get_by_text('NEXUS_SHELL_OK',exact=False).first).to_be_visible(timeout=5000)
            close_shell=page.get_by_label('Close Project Shell');
            if close_shell.count(): close_shell.first.click()
            page.wait_for_timeout(100); state=page.evaluate('window.__NEXUS_E2E_STATE__'); assert 'rt-shell-1' in state['stoppedRuntimes']

            page.keyboard.press('Control+K'); page.get_by_text('Open Composer',exact=True).click()
            page.get_by_title('Desktop windows').click(); expect(page.locator('.nx-desktop-workspace')).to_be_visible(); expect(page.get_by_text('COMPOSER',exact=True)).to_be_visible()
            page.get_by_title('Tabs / splits').click(); expect(page.get_by_text('COMPOSER',exact=True)).to_be_visible()

            composer=page.locator('.nx-composer-surface'); composer.get_by_role('button',name='Prepare Context').click(); expect(page.get_by_text('Context READY',exact=True)).to_be_visible(timeout=5000)
            composer.locator('textarea').first.fill('Implement the canonical deterministic Flow')
            turn=composer.get_by_role('button',name='Turn into Flow'); expect(turn).to_be_enabled(); turn.click()
            expect(page.get_by_text('Create Flow Draft',exact=True)).to_be_visible(); page.get_by_role('button',name='Generate Flow Draft').click()
            for wave in ['Wave 1','Wave 2','Wave 3']: expect(page.get_by_text(wave,exact=True)).to_be_visible(timeout=6000)
            for label in ['A · Architecture','B · Backend','C · Frontend','D · Verification']: expect(page.get_by_text(label,exact=True).first).to_be_visible()
            state=page.evaluate('window.__NEXUS_E2E_STATE__'); assert not any(path.endswith('/run') for _,path in state['writes']), 'Draft caused execution side effect'

            page.get_by_role('button',name='Approve & Run').click(); expect(page.get_by_text('FLOW RUN',exact=True)).to_be_visible(timeout=8000)
            expect(page.get_by_text('COMPLETED',exact=True).first).to_be_visible(); expect(page.get_by_text('Capsule · 2 receipt inputs',exact=True)).to_be_visible()
            dcard=page.locator('.nx-flow-run-step').filter(has_text='D · Verification'); expect(dcard).to_be_visible(); dcard.locator('summary').click(); expect(dcard.get_by_text('D completed with deterministic evidence',exact=False)).to_be_visible()

            state=page.evaluate('window.__NEXUS_E2E_STATE__'); assert state['csrfFailures']==0; assert any(path.endswith('/run') for _,path in state['writes'])
            if errors: raise AssertionError('Browser errors: '+' | '.join(errors))
            page.screenshot(path=str(artifacts/'browser-e2e-pass.png'),full_page=True)
            return {'status':'PASS','tests':['composer-context-gate','ask-existing-agent','project-shell-lifecycle','tabs-desktop-identity','flow-draft-no-side-effects','A-B||C-D-canvas','approve-and-run','receipt-handoff-ui','csrf-on-writes'], 'state':state}
        except Exception:
            page.screenshot(path=str(artifacts/'browser-e2e-failure.png'),full_page=True)
            (artifacts/'browser-e2e-failure.txt').write_text(traceback.format_exc()+'\n'+'\n'.join(errors))
            raise
        finally: browser.close()


def main() -> int:
    parser=argparse.ArgumentParser(); parser.add_argument('--artifacts',default=str(ROOT/'.e2e-artifacts')); args=parser.parse_args()
    if not (DIST/'bundle.js').exists(): raise SystemExit('web/dist missing; run npm --prefix web run build')
    result=run_e2e(Path(args.artifacts)); print(json.dumps(result,indent=2)); return 0

if __name__=='__main__': raise SystemExit(main())
