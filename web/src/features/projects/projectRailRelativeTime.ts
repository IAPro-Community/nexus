/** Format last_opened_at as a short relative time for the project rail. */
export function formatOpenedAgo(
  iso: string | undefined,
  locale: string,
  neverLabel: string,
): string {
  if (!iso) return neverLabel;
  const then = Date.parse(iso);
  if (Number.isNaN(then)) return neverLabel;
  const diffSec = Math.round((then - Date.now()) / 1000);
  const abs = Math.abs(diffSec);
  const rtf = new Intl.RelativeTimeFormat(locale, { numeric: 'auto' });
  if (abs < 60) return rtf.format(diffSec, 'second');
  const diffMin = Math.round(diffSec / 60);
  if (Math.abs(diffMin) < 60) return rtf.format(diffMin, 'minute');
  const diffHour = Math.round(diffMin / 60);
  if (Math.abs(diffHour) < 48) return rtf.format(diffHour, 'hour');
  const diffDay = Math.round(diffHour / 24);
  if (Math.abs(diffDay) < 30) return rtf.format(diffDay, 'day');
  const diffMonth = Math.round(diffDay / 30);
  if (Math.abs(diffMonth) < 12) return rtf.format(diffMonth, 'month');
  return rtf.format(Math.round(diffMonth / 12), 'year');
}
