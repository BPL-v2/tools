export function formatTime(ms: number) {
  return new Date(ms).toLocaleTimeString();
}

export function formatDateTime(ms: number) {
  return new Date(ms).toLocaleString();
}
