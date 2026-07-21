<script lang="ts">
	import { onMount } from 'svelte';
	import { Code, ConnectError } from '@connectrpc/connect';
	import { radioClient } from '$lib/connect/client';
	import type { SessionInfo, SessionArchiveInfo } from '$lib/proto/radio-jockey_pb';
	import LoadingButton from '$lib/components/LoadingButton.svelte';
	import EntryList from '$lib/components/EntryList.svelte';
	import { formatTimestamp } from '$lib/format';
	import { friendlyError } from '$lib/errors';

	const TOKEN_KEY = 'adminAuthToken';

	let authToken = $state('');
	let nonce = $state('');
	let passKey = $state('');
	let sessions = $state<SessionInfo[]>([]);
	let archives = $state<SessionArchiveInfo[]>([]);
	let error = $state('');
	let requestingNonce = $state(false);
	let submittingPassKey = $state(false);
	let stoppingIds = $state<Set<string>>(new Set());
	let deletingArchiveIds = $state<Set<bigint>>(new Set());
	let loaded = $state(false);

	async function loadAll() {
		await Promise.all([loadSessions(), loadArchives()]);
		loaded = true;
	}

	onMount(() => {
		authToken = localStorage.getItem(TOKEN_KEY) ?? '';
		if (authToken) {
			loadAll();
		}
	});

	async function loadSessions() {
		try {
			const res = await radioClient.listSessions({});
			sessions = res.sessions;
		} catch (err) {
			error = friendlyError(err);
		}
	}

	async function loadArchives() {
		try {
			const res = await radioClient.listSessionArchives({});
			archives = res.archives;
		} catch (err) {
			error = friendlyError(err);
		}
	}

	async function handleRequestNonce() {
		error = '';
		requestingNonce = true;
		try {
			const res = await radioClient.requestNonce({});
			nonce = res.nonce;
		} catch (err) {
			error = friendlyError(err);
		}
		requestingNonce = false;
	}

	async function handleSubmitPassKey(e: SubmitEvent) {
		e.preventDefault();
		error = '';
		submittingPassKey = true;
		try {
			const res = await radioClient.respondNonce({ passKey });
			authToken = res.authToken;
			localStorage.setItem(TOKEN_KEY, authToken);
			nonce = '';
			passKey = '';
			await loadAll();
		} catch (err) {
			error = friendlyError(err);
		}
		submittingPassKey = false;
	}

	function logout() {
		authToken = '';
		sessions = [];
		archives = [];
		localStorage.removeItem(TOKEN_KEY);
	}

	function handleAuthError(err: unknown) {
		if (err instanceof ConnectError && err.code === Code.Unauthenticated) {
			logout();
		} else {
			error = friendlyError(err);
		}
	}

	async function handleStop(sessionId: string) {
		error = '';
		stoppingIds = new Set(stoppingIds).add(sessionId);
		try {
			await radioClient.deleteSessionAuth(
				{ sessionId },
				{ headers: { authorization: `Bearer ${authToken}` } }
			);
			sessions = sessions.filter((s) => s.sessionId !== sessionId);
		} catch (err) {
			handleAuthError(err);
		}
		stoppingIds = new Set(stoppingIds);
		stoppingIds.delete(sessionId);
	}

	async function handleDeleteArchive(id: bigint) {
		error = '';
		deletingArchiveIds = new Set(deletingArchiveIds).add(id);
		try {
			await radioClient.deleteSessionArchive(
				{ id },
				{ headers: { authorization: `Bearer ${authToken}` } }
			);
			archives = archives.filter((a) => a.id !== id);
		} catch (err) {
			handleAuthError(err);
		}
		deletingArchiveIds = new Set(deletingArchiveIds);
		deletingArchiveIds.delete(id);
	}
</script>

{#if !authToken}
	<div class="panel">
		<h2>admin login</h2>
		<p>
			Run <code>go run ./cmd/signnonce &lt;nonce&gt;</code> from <code>server/</code> (or
			<code>just signnonce &lt;nonce&gt;</code>) with <code>PRIVATE_PASETO_KEY</code> set, then paste
			the output below.
		</p>

		<LoadingButton onclick={handleRequestNonce} loading={requestingNonce} label="request nonce" />
		{#if nonce}
			<p>nonce: <code>{nonce}</code></p>
		{/if}
	</div>

	<form onsubmit={handleSubmitPassKey}>
		<label for="passKey">signed passkey</label>
		<input id="passKey" type="text" bind:value={passKey} disabled={submittingPassKey} />
		<LoadingButton type="submit" loading={submittingPassKey} label="submit" />
	</form>
{:else}
	<div class="panel">
		<h2>live stations</h2>
		<button onclick={logout}>log out</button>
		{#if !loaded}
			<p>loading...</p>
		{:else}
			<EntryList items={sessions} emptyMessage="No active stations." key={(s) => s.sessionId}>
				{#snippet item(session)}
					<li>
						{session.sessionId}
						<LoadingButton
							onclick={() => handleStop(session.sessionId)}
							loading={stoppingIds.has(session.sessionId)}
							label="stop"
						/>
					</li>
				{/snippet}
			</EntryList>
		{/if}
	</div>

	<div class="panel">
		<h2>archived stations</h2>
		{#if !loaded}
			<p>loading...</p>
		{:else}
			<EntryList items={archives} emptyMessage="No archived stations." key={(a) => a.id}>
				{#snippet item(archive)}
					<li>
						<a href="/archive/{archive.id}"
							>{archive.sessionId} — {formatTimestamp(archive.createdAt)}</a
						>
						<LoadingButton
							onclick={() => handleDeleteArchive(archive.id)}
							loading={deletingArchiveIds.has(archive.id)}
							label="delete"
						/>
					</li>
				{/snippet}
			</EntryList>
		{/if}
	</div>
{/if}

{#if error}
	<p>{error}</p>
{/if}
