export type PtyLiveChrome = {
  title: string;
  questionnaire: boolean;
};

// Construct this expression from runtime control characters so ESLint does not
// mistake the protocol delimiters for accidental control characters.
const ESC = String.fromCharCode(0x1b);
const BEL = String.fromCharCode(0x07);
const OSC_TITLE = new RegExp(`${ESC}\\][02];([^${BEL}${ESC}]+)(?:${BEL}|${ESC}\\\\)`, 'g');

const QUESTIONNAIRE =
  /\bask_question\b|question\s+\d+\s+of\s+\d+|pergunta\s+\d+\s+de\s+\d+|question[aá]rio|select all that apply|press\s+(?:space|espaço).{0,40}toggle|space to (?:select|toggle)/i;

const QUESTIONNAIRE_CLEAR = /\b(thinking\.{0,3}|generating|running command|analyzing|searching)\b/i;

export function extractOscTitle(raw: string): string {
  if (!raw) return '';
  let last = '';
  for (const match of raw.matchAll(OSC_TITLE)) {
    const value = (match[1] || '').trim();
    if (value) last = value;
  }
  return last;
}

export function looksLikeQuestionnaire(text: string): boolean {
  return Boolean(text && QUESTIONNAIRE.test(text));
}

export function consumePtyOutputForChrome(raw: string, current: PtyLiveChrome): PtyLiveChrome {
  const oscTitle = extractOscTitle(raw);
  const title = oscTitle || current.title;
  let questionnaire = current.questionnaire;
  if (looksLikeQuestionnaire(raw) || looksLikeQuestionnaire(title)) {
    questionnaire = true;
  } else if (oscTitle && !looksLikeQuestionnaire(oscTitle)) {
    questionnaire = false;
  } else if (QUESTIONNAIRE_CLEAR.test(raw)) {
    questionnaire = false;
  }
  return { title, questionnaire };
}

export function ptyWindowHeading(input: {
  customTitle?: string;
  identity: string;
  liveTitle?: string;
  allowAiTitle?: boolean;
}): { heading: string; identityHint: string } {
  const customTitle = (input.customTitle || '').trim();
  const identity = (input.identity || '').trim();
  const live = (input.liveTitle || '').trim();
  // A custom identity and AI-driven title are independent values. When AI
  // updates are enabled (default), its live title is primary. When disabled,
  // a custom name becomes primary while the AI title remains visible as hint.
  // Without a custom name, the live title is always useful as the heading.
  if (live && (input.allowAiTitle !== false || !customTitle)) {
    const hint = customTitle || identity;
    return { heading: live, identityHint: hint && hint !== live ? hint : '' };
  }
  if (customTitle) {
    return { heading: customTitle, identityHint: live && live !== customTitle ? live : '' };
  }
  return { heading: identity || 'Terminal', identityHint: '' };
}
