
import root from '../root.js';
import { set_building, set_prerendering } from '__sveltekit/environment';
import { set_assets } from '$app/paths/internal/server';
import { set_manifest, set_read_implementation } from '__sveltekit/server';
import { set_private_env, set_public_env } from '../../../node_modules/@sveltejs/kit/src/runtime/shared-server.js';

export const options = {
	app_template_contains_nonce: false,
	async: false,
	csp: {"mode":"auto","directives":{"upgrade-insecure-requests":false,"block-all-mixed-content":false},"reportOnly":{"upgrade-insecure-requests":false,"block-all-mixed-content":false}},
	csrf_check_origin: true,
	csrf_trusted_origins: [],
	embedded: false,
	env_public_prefix: 'PUBLIC_',
	env_private_prefix: '',
	hash_routing: false,
	hooks: null, // added lazily, via `get_hooks`
	preload_strategy: "modulepreload",
	root,
	service_worker: false,
	service_worker_options: undefined,
	server_error_boundaries: false,
	templates: {
		app: ({ head, body, assets, nonce, env }) => "<!doctype html>\n<html lang=\"en\">\n\t<head>\n\t\t<meta charset=\"utf-8\" />\n\t\t<meta name=\"viewport\" content=\"width=device-width, initial-scale=1\" />\n\t\t<title>radio jockey</title>\n\t\t" + head + "\n\t</head>\n\t<body data-sveltekit-preload-data=\"hover\">\n\t\t<div class=\"bg-grid\" aria-hidden=\"true\"></div>\n\t\t<div class=\"content\">" + body + "</div>\n\t</body>\n</html>\n\n<style>\n\thtml {\n\t\tfont-family: 'Courier New', monospace;\n\t\tfont-weight: bold;\n\t}\n\n\th1 {\n\t\tdisplay: flex;\n\t\tflex-wrap: wrap;\n\t\talign-items: center;\n\t\tjustify-content: center;\n\t\tgap: 6px 8px;\n\t\tfont-size: 40px;\n\t}\n\n\t.wave-1 {\n\t\tfont-size: 22px;\n\t}\n\n\t.wave-2 {\n\t\tfont-size: 16px;\n\t}\n\n\t.wave-3 {\n\t\tfont-size: 11px;\n\t}\n\n\t@media (max-width: 600px) {\n\t\th1 {\n\t\t\tfont-size: 24px;\n\t\t}\n\n\t\t.wave-1 {\n\t\t\tfont-size: 17px;\n\t\t}\n\n\t\t.wave-2 {\n\t\t\tfont-size: 12px;\n\t\t}\n\n\t\t.wave-3 {\n\t\t\tfont-size: 8px;\n\t\t}\n\n\t\th2 {\n\t\t\tfont-size: 20px;\n\t\t}\n\n\t\th3 {\n\t\t\tfont-size: 18px;\n\t\t}\n\t}\n\n\tp,\n\tul,\n\tol,\n\tlabel {\n\t\tfont-size: 16px;\n\t}\n\n\tbody {\n\t\tmargin: 16px;\n\t\tbackground-color: #01026f;\n\t\tcolor: white;\n\t\tposition: relative;\n\t\tline-height: 24px;\n\t}\n\n\t.bg-grid {\n\t\tposition: fixed;\n\t\tinset: 0;\n\t\tbackground-image: url('/radio.png');\n\t\tbackground-size: 80px 80px;\n\t\tbackground-repeat: space;\n\t\topacity: 0.3;\n\t\tpointer-events: none;\n\t\tz-index: 0;\n\t}\n\n\t.content {\n\t\tposition: relative;\n\t\tz-index: 1;\n\t\t/* svh (not vh) — vh is measured with mobile browser chrome hidden, which overshoots\n\t\t   the actually-visible height when the address bar is shown, pushing the footer\n\t\t   past the fold */\n\t\tmin-height: calc(100svh - 32px);\n\t\tbox-sizing: border-box;\n\t\tborder: 1px solid white;\n\t\tpadding: 16px 32px;\n\t\tdisplay: flex;\n\t\tflex-direction: column;\n\t}\n\n\ta,\n\ta:visited,\n\ta:hover,\n\ta:active {\n\t\tcolor: inherit;\n\t\ttext-decoration: none;\n\t}\n\n\ta:hover {\n\t\ttext-decoration: underline;\n\t}\n\n\th2,\n\th3 {\n\t\tfont-family: 'Courier New', monospace;\n\t}\n\n\tp {\n\t\tmargin: 12px 0;\n\t}\n\n\t.panel {\n\t\tborder: 1px solid white;\n\t\tpadding: 16px;\n\t\tmargin-bottom: 24px;\n\t}\n\n\tform {\n\t\tborder: 1px solid white;\n\t\tpadding: 16px;\n\t\tmargin-bottom: 24px;\n\t\tdisplay: flex;\n\t\tflex-direction: column;\n\t\tgap: 12px;\n\t}\n\n\tul,\n\tol {\n\t\tlist-style: none;\n\t\tborder: 1px solid white;\n\t\tpadding: 0;\n\t\tmargin: 0 0 24px 0;\n\t}\n\n\tli {\n\t\tdisplay: flex;\n\t\tflex-wrap: wrap;\n\t\talign-items: center;\n\t\tgap: 8px;\n\t\tpadding: 16px;\n\t}\n\n\tli:not(:last-child) {\n\t\tbox-shadow: 0 1px 0 white;\n\t}\n\n\tol {\n\t\tcounter-reset: list-counter;\n\t}\n\n\tol > li {\n\t\tcounter-increment: list-counter;\n\t}\n\n\tol > li::before {\n\t\tcontent: counter(list-counter) '.';\n\t}\n\n\tbutton,\n\tinput {\n\t\tfont-family: inherit;\n\t\tfont-size: 16px;\n\t\tfont-weight: bold;\n\t\tbackground: transparent;\n\t\tcolor: white;\n\t\tborder: 1px solid white;\n\t\tpadding: 6px 10px;\n\t}\n\n\tbutton:hover {\n\t\ttext-decoration: underline;\n\t}\n</style>\n",
		error: ({ status, message }) => "<!doctype html>\n<html lang=\"en\">\n\t<head>\n\t\t<meta charset=\"utf-8\" />\n\t\t<title>" + message + "</title>\n\n\t\t<style>\n\t\t\tbody {\n\t\t\t\t--bg: white;\n\t\t\t\t--fg: #222;\n\t\t\t\t--divider: #ccc;\n\t\t\t\tbackground: var(--bg);\n\t\t\t\tcolor: var(--fg);\n\t\t\t\tfont-family:\n\t\t\t\t\tsystem-ui,\n\t\t\t\t\t-apple-system,\n\t\t\t\t\tBlinkMacSystemFont,\n\t\t\t\t\t'Segoe UI',\n\t\t\t\t\tRoboto,\n\t\t\t\t\tOxygen,\n\t\t\t\t\tUbuntu,\n\t\t\t\t\tCantarell,\n\t\t\t\t\t'Open Sans',\n\t\t\t\t\t'Helvetica Neue',\n\t\t\t\t\tsans-serif;\n\t\t\t\tdisplay: flex;\n\t\t\t\talign-items: center;\n\t\t\t\tjustify-content: center;\n\t\t\t\theight: 100vh;\n\t\t\t\tmargin: 0;\n\t\t\t}\n\n\t\t\t.error {\n\t\t\t\tdisplay: flex;\n\t\t\t\talign-items: center;\n\t\t\t\tmax-width: 32rem;\n\t\t\t\tmargin: 0 1rem;\n\t\t\t}\n\n\t\t\t.status {\n\t\t\t\tfont-weight: 200;\n\t\t\t\tfont-size: 3rem;\n\t\t\t\tline-height: 1;\n\t\t\t\tposition: relative;\n\t\t\t\ttop: -0.05rem;\n\t\t\t}\n\n\t\t\t.message {\n\t\t\t\tborder-left: 1px solid var(--divider);\n\t\t\t\tpadding: 0 0 0 1rem;\n\t\t\t\tmargin: 0 0 0 1rem;\n\t\t\t\tmin-height: 2.5rem;\n\t\t\t\tdisplay: flex;\n\t\t\t\talign-items: center;\n\t\t\t}\n\n\t\t\t.message h1 {\n\t\t\t\tfont-weight: 400;\n\t\t\t\tfont-size: 1em;\n\t\t\t\tmargin: 0;\n\t\t\t}\n\n\t\t\t@media (prefers-color-scheme: dark) {\n\t\t\t\tbody {\n\t\t\t\t\t--bg: #222;\n\t\t\t\t\t--fg: #ddd;\n\t\t\t\t\t--divider: #666;\n\t\t\t\t}\n\t\t\t}\n\t\t</style>\n\t</head>\n\t<body>\n\t\t<div class=\"error\">\n\t\t\t<span class=\"status\">" + status + "</span>\n\t\t\t<div class=\"message\">\n\t\t\t\t<h1>" + message + "</h1>\n\t\t\t</div>\n\t\t</div>\n\t</body>\n</html>\n"
	},
	version_hash: "v6sxw6"
};

export async function get_hooks() {
	let handle;
	let handleFetch;
	let handleError;
	let handleValidationError;
	let init;
	

	let reroute;
	let transport;
	

	return {
		handle,
		handleFetch,
		handleError,
		handleValidationError,
		init,
		reroute,
		transport
	};
}

export { set_assets, set_building, set_manifest, set_prerendering, set_private_env, set_public_env, set_read_implementation };
