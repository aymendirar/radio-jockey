<script lang="ts">
	import { radioClient } from '$lib/connect/client';
	import { friendlyError, addTrackErrorMessage } from '$lib/errors';
	import type { SearchResult } from '$lib/youtube';
	import YouTubeSearchBar from './YouTubeSearchBar.svelte';
	import YouTubeSearchResults from './YouTubeSearchResults.svelte';

	let { sessionId, onAdded }: { sessionId: string; onAdded: () => void } = $props();

	let currentQuery = $state('');
	let results = $state<SearchResult[]>([]);
	let nextPageToken = $state<string | undefined>(undefined);
	// tokens actually used to fetch each page so far (index 0 = undefined, page 1's token)
	let pageTokens = $state<(string | undefined)[]>([undefined]);
	let pageIndex = $state(0);
	let searching = $state(false);
	let pageLoadingDirection = $state<'next' | 'prev' | null>(null);
	let searchError = $state('');
	let addingIds = $state<Set<string>>(new Set());
	let addError = $state('');

	function resetPagination() {
		results = [];
		nextPageToken = undefined;
		pageTokens = [undefined];
		pageIndex = 0;
	}

	async function runSearch(query: string, page?: { token?: string; direction: 'next' | 'prev' }) {
		if (page) {
			pageLoadingDirection = page.direction;
		} else {
			searching = true;
		}
		searchError = '';

		try {
			const params = new URLSearchParams({ q: query });
			if (page?.token) params.set('pageToken', page.token);
			const res = await fetch(`/api/youtube-search?${params}`);

			if (!res.ok) {
				if (res.status === 503) {
					searchError = 'YouTube search is not configured.';
				} else if (res.status === 429) {
					searchError =
						'Daily YouTube search limit reached. Try again tomorrow, or paste a YouTube URL below instead.';
				} else {
					searchError = 'Search failed. Please try again.';
				}
				// a fresh search failing has nothing worth keeping; a failed page
				// turn should leave the current (still valid) page in place
				if (!page) {
					resetPagination();
				}
			} else {
				const data = await res.json();
				if (data.cached) {
					console.log('youtube search cache hit', query, page?.token);
				}
				results = data.results;
				nextPageToken = data.nextPageToken;
			}
		} catch (err) {
			searchError = friendlyError(err);
			if (!page) {
				resetPagination();
			}
		}

		if (page) {
			pageLoadingDirection = null;
		} else {
			searching = false;
		}
	}

	function handleSearch(query: string) {
		currentQuery = query;
		resetPagination();
		runSearch(query);
	}

	function handleNext() {
		if (!nextPageToken) return;
		const token = nextPageToken;
		pageTokens = [...pageTokens.slice(0, pageIndex + 1), token];
		pageIndex += 1;
		runSearch(currentQuery, { token, direction: 'next' });
	}

	function handlePrev() {
		if (pageIndex === 0) return;
		pageIndex -= 1;
		runSearch(currentQuery, { token: pageTokens[pageIndex], direction: 'prev' });
	}

	async function handleAdd(videoId: string) {
		addError = '';
		addingIds = new Set(addingIds).add(videoId);
		try {
			await radioClient.addTrack({
				sessionId,
				trackUrl: `https://www.youtube.com/watch?v=${videoId}`
			});
			onAdded();
		} catch (err) {
			addError = addTrackErrorMessage(err);
		}
		addingIds = new Set(addingIds);
		addingIds.delete(videoId);
	}
</script>

<div class="panel">
	<h3>search youtube</h3>
	<p class="subtitle">limited to ~100 searches/day across all listeners</p>
	<YouTubeSearchBar onSearch={handleSearch} {searching} />

	{#if searchError}
		<p>{searchError}</p>
	{/if}

	{#if currentQuery && !searchError && !searching}
		<YouTubeSearchResults
			{results}
			{addingIds}
			onAdd={handleAdd}
			onNext={handleNext}
			onPrev={handlePrev}
			hasNext={!!nextPageToken}
			hasPrev={pageIndex > 0}
			{pageLoadingDirection}
		/>
	{/if}

	{#if addError}
		<p>{addError}</p>
	{/if}
</div>

<style>
	.subtitle {
		font-size: 14px;
		opacity: 0.7;
		margin-top: -6px;
	}
</style>
