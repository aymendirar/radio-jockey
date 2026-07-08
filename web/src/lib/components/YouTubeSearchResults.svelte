<script lang="ts">
	import EntryList from './EntryList.svelte';
	import TrackListItem from './TrackListItem.svelte';
	import LoadingButton from './LoadingButton.svelte';
	import type { SearchResult } from '$lib/youtube';

	let {
		results,
		addingIds,
		onAdd,
		onNext,
		onPrev,
		hasNext,
		hasPrev,
		pageLoading
	}: {
		results: SearchResult[];
		addingIds: Set<string>;
		onAdd: (videoId: string) => void;
		onNext: () => void;
		onPrev: () => void;
		hasNext: boolean;
		hasPrev: boolean;
		pageLoading: boolean;
	} = $props();
</script>

<EntryList items={results} emptyMessage="No results." key={(r) => r.videoId}>
	{#snippet item(result)}
		<TrackListItem
			track={{ title: result.title, artist: result.channelTitle, albumArtUrl: result.thumbnailUrl }}
			href={`https://www.youtube.com/watch?v=${result.videoId}`}
		>
			<LoadingButton
				onclick={() => onAdd(result.videoId)}
				loading={addingIds.has(result.videoId)}
				label="add"
			/>
		</TrackListItem>
	{/snippet}
</EntryList>

{#if results.length > 0}
	<div class="pagination">
		<LoadingButton onclick={onPrev} loading={pageLoading} disabled={!hasPrev} label="< prev" />
		<LoadingButton onclick={onNext} loading={pageLoading} disabled={!hasNext} label="next >" />
	</div>
{/if}

<style>
	.pagination {
		display: flex;
		justify-content: space-between;
		gap: 8px;
	}
</style>
