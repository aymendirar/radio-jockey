<script lang="ts">
	import { onMount } from 'svelte';
	import { page } from '$app/state';
	import { Code, ConnectError } from '@connectrpc/connect';
	import { radioClient } from '$lib/connect/client';
	import type { Track } from '$lib/proto/radio-jockey_pb';
	import TrackListItem from '$lib/components/TrackListItem.svelte';
	import NotFound from '$lib/components/NotFound.svelte';
	import { formatTimestamp } from '$lib/format';

	const archiveId = BigInt(page.params.archiveId!);

	let notFound = $state(false);
	let sessionId = $state('');
	let createdAt = $state<bigint>(0n);
	let tracks = $state<Track[]>([]);
	let error = $state('');

	onMount(async () => {
		try {
			const res = await radioClient.getSessionArchive({ id: archiveId });
			sessionId = res.sessionId;
			createdAt = res.createdAt;
			tracks = res.tracks;
		} catch (err) {
			if (err instanceof ConnectError && err.code === Code.NotFound) {
				notFound = true;
				return;
			}
			error = err instanceof Error ? err.message : String(err);
		}
	});
</script>

{#if notFound}
	<NotFound message="This archive doesn't exist." backHref="/archive" backLabel="back to archive" />
{:else}
	<h2>{sessionId}</h2>
	<p>{formatTimestamp(createdAt)}</p>

	<ol>
		{#each tracks as track (track.id)}
			<TrackListItem
				{track}
				href={track.source === 'youtube'
					? `https://www.youtube.com/watch?v=${track.sourceId}`
					: undefined}
			/>
		{/each}
	</ol>
{/if}

{#if error}
	<p>{error}</p>
{/if}
