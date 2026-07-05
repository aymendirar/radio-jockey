export function formatTimestamp(unixSeconds: bigint): string {
	const date = new Date(Number(unixSeconds) * 1000);
	const pad = (n: number) => String(n).padStart(2, '0');
	const datePart = `${date.getUTCFullYear()}-${pad(date.getUTCMonth() + 1)}-${pad(date.getUTCDate())}`;
	const timePart = `${pad(date.getUTCHours())}:${pad(date.getUTCMinutes())}:${pad(date.getUTCSeconds())}`;
	return `${datePart}-${timePart}Z`;
}
