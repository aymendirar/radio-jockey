<script lang="ts">
	let count = $state<number | null>(null);

	$effect(() => {
		let unsubscribe: (() => void) | undefined;

		(async () => {
			const { playhtml } = await import('playhtml');
			await playhtml.ready;
			playhtml.presence.setMyPresence('online', true);
			unsubscribe = playhtml.presence.onPresenceChange('online', (presences) => {
				count = presences.size;
			});
		})();

		return () => unsubscribe?.();
	});
</script>

<span class="visitor-count" class:invisible={count === null}>
	{count ?? 0} {count === 1 ? 'person' : 'people'} online
</span>

<style>
	.visitor-count {
		flex-basis: 100%;
		text-align: center;
		font-size: 0.75rem;
		opacity: 0.6;
	}

	.invisible {
		visibility: hidden;
	}
</style>
