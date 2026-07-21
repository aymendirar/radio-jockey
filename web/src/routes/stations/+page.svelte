<script lang="ts">
	import { onMount } from 'svelte';
	import { radioClient } from '$lib/connect/client';
	import type { SessionInfo } from '$lib/proto/radio-jockey_pb';
	import EntryList from '$lib/components/EntryList.svelte';
	import { friendlyError } from '$lib/errors';

	let sessions = $state<SessionInfo[]>([]);
	let error = $state('');
	let loaded = $state(false);

	onMount(async () => {
		try {
			const res = await radioClient.listSessions({});
			sessions = res.sessions;
		} catch (err) {
			error = friendlyError(err);
		}
		loaded = true;
	});
</script>

<p><a href="/stations/create">create a new station</a></p>

{#if !loaded}
	<p>loading...</p>
{:else}
	<EntryList
		items={sessions}
		emptyMessage="No live stations."
		key={(s) => s.sessionId}
		class="arrow-list"
	>
		{#snippet item(session)}
			<li><a href="/stations/{session.sessionId}">{session.sessionId}</a></li>
		{/snippet}
	</EntryList>
{/if}

{#if error}
	<p>{error}</p>
{/if}
