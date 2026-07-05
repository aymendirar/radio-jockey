<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import { page } from '$app/state';
	import { Code, ConnectError } from '@connectrpc/connect';
	import { radioClient } from '$lib/connect/client';
	import type { Track } from '$lib/proto/radio-jockey_pb';
	import TrackListItem from '$lib/components/TrackListItem.svelte';
	import LoadingButton from '$lib/components/LoadingButton.svelte';
	import NotFound from '$lib/components/NotFound.svelte';

	const sessionId = page.params.sessionId!;

	let audioEl: HTMLAudioElement;
	let playing = $state(false);
	let volume = $state(1);

	function play() {
		if (!streamUrl) return;
		audioEl.src = streamUrl;
		audioEl.load();
		audioEl.play();
		playing = true;
	}

	function stop() {
		audioEl.pause();
		audioEl.removeAttribute('src');
		audioEl.load();
		playing = false;
	}

	$effect(() => {
		const v = volume;
		if (audioEl) audioEl.volume = v;
	});

	let notFound = $state(false);
	let streamUrl = $state('');
	let tracks = $state<Track[]>([]);
	let trackUrl = $state('');
	let addError = $state('');
	let adding = $state(false);
	let generalError = $state('');
	let refreshing = $state(false);
	let skipping = $state(false);
	let removingIndices = $state<Set<number>>(new Set());
	let queueLoaded = $state(false);

	let pollHandle: ReturnType<typeof setInterval> | undefined;

	async function fetchQueue() {
		try {
			const res = await radioClient.listQueue({ sessionId });
			tracks = res.tracks;
		} catch (err) {
			generalError = err instanceof Error ? err.message : String(err);
		}
	}

	onMount(async () => {
		try {
			const res = await radioClient.getSession({ sessionId });
			streamUrl = res.streamUrl;
		} catch (err) {
			if (err instanceof ConnectError && err.code === Code.NotFound) {
				notFound = true;
				return;
			}
			generalError = err instanceof Error ? err.message : String(err);
			return;
		}

		await fetchQueue();
		queueLoaded = true;
		pollHandle = setInterval(fetchQueue, 5000);
	});

	onDestroy(() => {
		if (pollHandle) clearInterval(pollHandle);
	});

	async function handleAddTrack(e: SubmitEvent) {
		e.preventDefault();
		addError = '';
		adding = true;
		try {
			await radioClient.addTrack({ sessionId, trackUrl });
			trackUrl = '';
			await fetchQueue();
		} catch (err) {
			if (err instanceof ConnectError) {
				switch (err.code) {
					case Code.InvalidArgument:
						addError = 'Invalid URL. Please try again with a YouTube link!';
						break;
					case Code.NotFound:
						addError = 'That video is unavailable or the station has ended.';
						break;
					case Code.ResourceExhausted:
						addError = 'The queue is full!';
						break;
					default:
						addError = 'Something went wrong adding that track.';
				}
			} else {
				addError = err instanceof Error ? err.message : String(err);
			}
		}
		adding = false;
	}

	async function handleRefresh() {
		refreshing = true;
		await fetchQueue();
		refreshing = false;
	}

	async function handleSkip() {
		skipping = true;
		try {
			await radioClient.skipTrack({ sessionId });
			await fetchQueue();
		} catch (err) {
			generalError = err instanceof Error ? err.message : String(err);
		}
		skipping = false;
	}

	async function handleRemove(index: number) {
		removingIndices = new Set(removingIndices).add(index);
		try {
			await radioClient.removeTrack({ sessionId, index });
			await fetchQueue();
		} catch (err) {
			generalError = err instanceof Error ? err.message : String(err);
		}
		removingIndices = new Set(removingIndices);
		removingIndices.delete(index);
	}
</script>

{#if notFound}
	<NotFound message="This station doesn't exist." backHref="/stations" backLabel="back to stations" />
{:else}
	<h2>{sessionId}</h2>

	<div class="panel">
		{#if streamUrl}
			<audio bind:this={audioEl}></audio>
			<div class="player-controls">
				{#if playing}
					<button class="btn-stop" onclick={stop} aria-label="stop">[ ]</button>
				{:else}
					<button class="btn-play" onclick={play} aria-label="play">&gt;</button>
				{/if}
				<input type="range" min="0" max="1" step="0.01" bind:value={volume} aria-label="volume" />
			</div>
		{/if}
	</div>

	<div class="panel">
		<h3>now playing</h3>
		{#if !queueLoaded}
			<p>loading...</p>
		{:else if tracks.length > 0}
			<div class="now-playing">
				{#if tracks[0].albumArtUrl}
					<img class="album-art" src={tracks[0].albumArtUrl} alt="" />
				{/if}
				<p>{tracks[0].title} — {tracks[0].artist}</p>
			</div>
		{:else}
			<p>nothing queued</p>
		{/if}
	</div>

	<div class="panel">
		<h3>
			queue
			<LoadingButton onclick={handleRefresh} loading={refreshing} label="refresh" />
		</h3>
		{#if !queueLoaded}
			<p>loading...</p>
		{:else}
			<ol>
				{#each tracks as track, i (track.id)}
					<TrackListItem {track}>
						{#if i > 0}
							<LoadingButton
								onclick={() => handleRemove(i)}
								loading={removingIndices.has(i)}
								label="remove"
							/>
						{/if}
					</TrackListItem>
				{/each}
			</ol>
		{/if}

		<LoadingButton onclick={handleSkip} loading={skipping} label="skip" />
	</div>

	<form onsubmit={handleAddTrack}>
		<label for="trackUrl">add a track</label>
		<input id="trackUrl" type="text" bind:value={trackUrl} disabled={adding} placeholder="YouTube URL" />
		<LoadingButton type="submit" loading={adding} label="add" />
	</form>

	{#if addError}
		<p>{addError}</p>
	{/if}
{/if}

{#if generalError}
	<p>{generalError}</p>
{/if}

<style>
	.player-controls {
		display: flex;
		align-items: center;
		gap: 12px;
	}

	.player-controls input[type='range'] {
		accent-color: white;
		border: none;
		padding: 0;
	}

	.btn-play,
	.btn-stop {
		width: 44px;
		height: 44px;
		padding: 0;
		display: flex;
		align-items: center;
		justify-content: center;
		font-family: 'Courier New', monospace;
		font-size: 1.1em;
		background: transparent;
		border-radius: 0;
	}

	.btn-play {
		border: 2px solid #2f9e1a;
		color: #2f9e1a;
	}

	.btn-stop {
		border: 2px solid #c0392b;
		color: #c0392b;
	}

	.now-playing {
		display: flex;
		align-items: center;
		gap: 12px;
	}

	.album-art {
		width: 120px;
		height: 90px;
		object-fit: cover;
		border: 1px solid white;
	}
</style>
