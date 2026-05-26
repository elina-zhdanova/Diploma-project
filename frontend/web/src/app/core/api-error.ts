/** Ответ API: { error: { code, message } } */
export function apiErrorMessage(err: unknown): string {
  const e = err as { error?: { error?: { message?: string } }; message?: string };
  return e?.error?.error?.message ?? e?.message ?? 'Ошибка запроса';
}
