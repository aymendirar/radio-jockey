export const GENERIC_ERROR = 'Something went wrong. Please try again.';

// ConnectError/fetch failures carry raw transport text (e.g. "[unknown] Load failed" on iOS
// Safari when an in-flight request gets killed) that's meaningless to a user — log it and
// show a generic message instead.
export function friendlyError(err: unknown): string {
	console.error(err);
	return GENERIC_ERROR;
}
