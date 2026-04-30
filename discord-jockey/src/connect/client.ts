import { createClient } from "@connectrpc/connect";
import { createConnectTransport } from "@connectrpc/connect-node";
import { RadioService } from "../proto/radio-jockey_pb.js";

const { SERVER_HOST, SERVER_PORT } = process.env;

const transport = createConnectTransport({
  httpVersion: "1.1",
  baseUrl: `http://${SERVER_HOST}:${SERVER_PORT}`,
});

export const radioClient = createClient(RadioService, transport);
