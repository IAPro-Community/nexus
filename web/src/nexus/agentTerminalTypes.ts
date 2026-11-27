/** Maestro skill as exposed by the agent-terminal skills drawer. */
export interface AgentTerminalSkill {
  id: string;
  name?: string;
  description?: string;
  category?: string;
  risk?: string;
  triggers?: string[];
  aliases?: string[];
}
