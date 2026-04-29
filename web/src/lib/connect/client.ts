import { createClient } from '@connectrpc/connect';
import { createConnectTransport } from '@connectrpc/connect-web';
import { RadioService } from '$lib/proto/radio-jockey_pb';

const transport = createConnectTransport({
	baseUrl: ''
});

export const radioClient = createClient(RadioService, transport);
