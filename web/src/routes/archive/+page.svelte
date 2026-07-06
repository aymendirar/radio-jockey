<script lang="ts">
	import { onMount } from 'svelte';
	import { radioClient } from '$lib/connect/client';
	import type { SessionArchiveInfo } from '$lib/proto/radio-jockey_pb';
	import EntryList from '$lib/components/EntryList.svelte';
	import { formatTimestamp } from '$lib/format';
	import { friendlyError } from '$lib/errors';

	let archives = $state<SessionArchiveInfo[]>([]);
	let error = $state('');
	let loaded = $state(false);

	onMount(async () => {
		try {
			const res = await radioClient.listSessionArchives({});
			archives = res.archives;
		} catch (err) {
			error = friendlyError(err);
		}
		loaded = true;
	});
</script>

<h2>station archive</h2>

{#if !loaded}
	<p>loading...</p>
{:else}
	<EntryList
		items={archives}
		emptyMessage="No archived stations."
		key={(a) => a.id}
		class="arrow-list"
	>
		{#snippet item(archive)}
			<li>
				<a href="/archive/{archive.id}"
					>{archive.sessionId} — {formatTimestamp(archive.createdAt)}</a
				>
			</li>
		{/snippet}
	</EntryList>
{/if}

{#if error}
	<p>{error}</p>
{/if}

<style>
	:global(.arrow-list > li > a::before) {
		content: '> ';
	}
</style>
