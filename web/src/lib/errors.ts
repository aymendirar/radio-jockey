import { Code, ConnectError } from '@connectrpc/connect';

export const GENERIC_ERROR = 'Something went wrong. Please try again.';

// ConnectError/fetch failures carry raw transport text (e.g. "[unknown] Load failed" on iOS
// Safari when an in-flight request gets killed) that's meaningless to a user — log it and
// show a generic message instead.
export function friendlyError(err: unknown): string {
	console.error(err);
	return GENERIC_ERROR;
}

// shared between the manual URL form and YouTube search results — both call AddTrack
export function addTrackErrorMessage(err: unknown): string {
	if (err instanceof ConnectError) {
		switch (err.code) {
			case Code.InvalidArgument:
				return 'Invalid URL. Please try again with a YouTube link!';
			case Code.NotFound:
				return 'That video is unavailable or the station has ended.';
			case Code.ResourceExhausted:
				// server reuses this code for both a full queue and a rate-limit hit
				// (see server/src/connect/interceptor.go); rawMessage disambiguates
				return err.rawMessage === 'rate limit exceeded'
					? "You're adding tracks too fast. Wait a moment and try again."
					: 'The queue is full!';
			default:
				return 'Something went wrong adding that track.';
		}
	}
	return friendlyError(err);
}
