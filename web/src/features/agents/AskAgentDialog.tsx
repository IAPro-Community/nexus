import React, { useState } from 'react';
import { Send } from 'lucide-react';
import { Button, Card, Dialog, Textarea } from '../../design-system';
import { nexus } from '../../nexus/api';
import type { Agent } from '../../types';
import { askAgentAction } from './askAgentModel';

export const AskAgentDialog: React.FC<{
  agent: Agent | null;
  onClose: () => void;
  onDone: () => Promise<void>;
}> = ({ agent, onClose, onDone }) => {
  const [instruction, setInstruction] = useState('');
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState('');

  const submit = async () => {
    if (!agent || !instruction.trim()) return;
    setBusy(true);
    setError('');
    try {
      if (askAgentAction(agent) === 'START_AND_ASK') await nexus.startAgent(agent.id);
      await nexus.askAgent(agent.id, instruction.trim());
      setInstruction('');
      onClose();
      await onDone();
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setBusy(false);
    }
  };

  return (
    <Dialog open={!!agent} onClose={onClose} title={agent ? `Ask ${agent.name}` : 'Ask Agent'} wide>
      <div className="nx-ask-agent-dialog">
        <p className="nx-muted-copy">Send a prompt to this existing Agent. This action never creates a new Agent.</p>
        <Textarea value={instruction} onChange={setInstruction} placeholder="What should this Agent do next?" rows={6} />
        {error && <Card className="nx-inline-error" role="alert">{error}</Card>}
        <div className="nx-dialog-actions">
          <Button onClick={onClose} disabled={busy}>Cancel</Button>
          <Button tone="brand" onClick={() => void submit()} disabled={!instruction.trim() || busy}>
            <Send size={14} /> {busy ? 'Sending…' : 'Send instruction'}
          </Button>
        </div>
      </div>
    </Dialog>
  );
};
