export function fmtDate(iso: string): string {
  return iso.replace("T", " ").slice(0, 16) + " UTC";
}

export function fmtDateTime(iso: string): string {
  return iso.replace("T", " ").slice(0, 19) + " UTC";
}
