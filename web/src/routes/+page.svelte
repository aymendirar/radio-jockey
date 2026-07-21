<div>
	<ul>
		<a href="/stations/create"><li>create a new station</li></a>
		<a href="/stations"><li>listen to live stations</li></a>
		<a href="/archive"><li>view station archive</li></a>
	</ul>
</div>

<footer>
	<span class="chat-hint">press <kbd>/</kbd> to chat with other listeners —{' '}</span>cursors powered
	by
	<a href="https://playhtml.fun" target="_blank" rel="noopener noreferrer">playhtml</a>
</footer>

<style>
	ul {
		list-style-type: '>  ';
	}

	footer {
		margin-top: auto;
		font-size: 0.8rem;
		opacity: 0.6;
		text-align: center;
	}

	kbd {
		border: 1px solid currentColor;
		border-radius: 3px;
		padding: 0 0.3em;
	}

	@media (max-width: 600px) {
		.chat-hint {
			display: none;
		}
	}
</style>
