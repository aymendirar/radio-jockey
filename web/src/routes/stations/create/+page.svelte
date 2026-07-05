<script lang="ts">
	import { goto } from '$app/navigation';
	import { Code, ConnectError } from '@connectrpc/connect';
	import { radioClient } from '$lib/connect/client';
	import LoadingButton from '$lib/components/LoadingButton.svelte';

	let name = $state('');
	let archive = $state(false);
	let error = $state('');
	let submitting = $state(false);

	function slugify(input: string): string {
		return input
			.toLowerCase()
			.replace(/[^a-z0-9]+/g, '-')
			.replace(/^-+|-+$/g, '');
	}

	async function handleSubmit(e: SubmitEvent) {
		e.preventDefault();
		const sessionId = slugify(name);
		if (!sessionId) {
			error = 'Please enter a station name.';
			return;
		}

		error = '';
		submitting = true;
		try {
			await radioClient.createSession({ sessionId, archive });
			await goto(`/stations/${sessionId}`);
		} catch (err) {
			if (err instanceof ConnectError && err.code === Code.AlreadyExists) {
				error = 'That name is taken, pick another.';
			} else {
				error = err instanceof Error ? err.message : String(err);
			}
		}
		submitting = false;
	}
</script>

<form onsubmit={handleSubmit}>
	<label for="name">station name</label>
	<input id="name" type="text" bind:value={name} disabled={submitting} />
	<label for="archive">
		<input id="archive" type="checkbox" bind:checked={archive} disabled={submitting} />
		archive this session
	</label>
	<LoadingButton type="submit" loading={submitting} label="create" />
</form>

{#if error}
	<p>{error}</p>
{/if}

<style>
	:global(button) {
		padding: 12px 24px;
	}
</style>
