/** Поля заявки для текстового поиска (логин, ФИО, роли доступа, обоснование, id). */
export interface RequestSearchFields {
  id?: string;
  initiator_name?: string | null;
  initiator_login?: string | null;
  access_role_names?: string | null;
  justification?: string | null;
}

/** Поиск без учёта регистра; несколько слов — каждое должно встретиться хотя бы в одном поле. */
export function matchesRequestSearch(r: RequestSearchFields, query: string): boolean {
  const norm = query.trim().toLowerCase();
  if (!norm) {
    return true;
  }
  const tokens = norm.split(/\s+/).filter(Boolean);
  const blob = [r.initiator_login, r.initiator_name, r.access_role_names, r.justification, r.id]
    .map((x) => String(x ?? '').toLowerCase())
    .join(' | ');
  return tokens.every((t) => blob.includes(t));
}
