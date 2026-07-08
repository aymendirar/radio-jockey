import { json } from '@sveltejs/kit';
import { env } from '$env/dynamic/private';
import type { RequestHandler } from './$types';
import type { SearchResult } from '$lib/youtube';

type YouTubeSearchItem = {
	id?: { videoId?: string };
	snippet: {
		title: string;
		channelTitle: string;
		thumbnails?: { default?: { url: string } };
	};
};

type YouTubeSearchResponse = {
	items?: YouTubeSearchItem[];
	nextPageToken?: string;
	prevPageToken?: string;
};

type CacheEntry = {
	results: SearchResult[];
	nextPageToken?: string;
	prevPageToken?: string;
	expiresAt: number;
};

// generous TTL — matches the YouTube API quota's own daily reset, so a repeat
// search within the same day never costs quota twice
const CACHE_TTL_MS = 24 * 60 * 60 * 1000;
const CACHE_MAX_ENTRIES = 1000;

const cache = new Map<string, CacheEntry>();

function cacheKey(q: string, pageToken: string | null): string {
	return `${q.toLowerCase()}::${pageToken ?? ''}`;
}

function setCache(key: string, entry: CacheEntry) {
	if (cache.size >= CACHE_MAX_ENTRIES) {
		const oldestKey = cache.keys().next().value;
		if (oldestKey !== undefined) cache.delete(oldestKey);
	}
	cache.set(key, entry);
}

// reads YouTube's structured error reason (e.g. "quotaExceeded") rather than
// substring-matching the raw body, so unrelated wording changes can't break detection
function parseErrorReason(errorText: string): string | undefined {
	try {
		const parsed = JSON.parse(errorText);
		return parsed?.error?.errors?.[0]?.reason;
	} catch {
		return undefined;
	}
}

export const GET: RequestHandler = async ({ url, fetch }) => {
	const q = url.searchParams.get('q')?.trim();
	if (!q) {
		return json({ error: 'missing query' }, { status: 400 });
	}

	const pageToken = url.searchParams.get('pageToken');
	const key = cacheKey(q, pageToken);

	const cached = cache.get(key);
	if (cached) {
		if (cached.expiresAt > Date.now()) {
			console.log('youtube search cache hit', key);
			return json({
				results: cached.results,
				nextPageToken: cached.nextPageToken,
				prevPageToken: cached.prevPageToken,
				cached: true
			});
		}
		cache.delete(key);
	}

	const apiKey = env.YOUTUBE_API_KEY;
	if (!apiKey) {
		return json({ error: 'search not configured' }, { status: 503 });
	}

	const ytUrl = new URL('https://www.googleapis.com/youtube/v3/search');
	ytUrl.searchParams.set('part', 'snippet');
	ytUrl.searchParams.set('type', 'video');
	ytUrl.searchParams.set('maxResults', '5');
	ytUrl.searchParams.set('q', q);
	ytUrl.searchParams.set('key', apiKey);
	if (pageToken) ytUrl.searchParams.set('pageToken', pageToken);

	const res = await fetch(ytUrl);
	if (!res.ok) {
		const errorText = await res.text();
		console.error('youtube search failed', res.status, errorText);
		if (res.status === 403) {
			const reason = parseErrorReason(errorText);
			if (reason === 'quotaExceeded' || reason === 'dailyLimitExceeded') {
				return json({ error: 'quota exceeded' }, { status: 429 });
			}
		}
		return json({ error: 'search failed' }, { status: 502 });
	}

	const data: YouTubeSearchResponse = await res.json();

	const results = (data.items ?? [])
		.map((item): SearchResult | null => {
			const videoId = item.id?.videoId;
			if (!videoId) return null;
			return {
				videoId,
				title: item.snippet.title,
				channelTitle: item.snippet.channelTitle,
				thumbnailUrl: item.snippet.thumbnails?.default?.url ?? ''
			};
		})
		.filter((r): r is SearchResult => r !== null);

	setCache(key, {
		results,
		nextPageToken: data.nextPageToken,
		prevPageToken: data.prevPageToken,
		expiresAt: Date.now() + CACHE_TTL_MS
	});

	return json({ results, nextPageToken: data.nextPageToken, prevPageToken: data.prevPageToken });
};
