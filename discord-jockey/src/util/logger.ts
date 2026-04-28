import { ConsoleTransport, LogLayer, StructuredTransport } from "loglayer";
import {
  getSimplePrettyTerminal,
  moonlight,
} from "@loglayer/transport-simple-pretty-terminal";

export const logger = new LogLayer({
  transport: [
    new ConsoleTransport({
      logger: console,
      enabled: process.env.NODE_ENV !== "development",
    }),
    getSimplePrettyTerminal({
      enabled: process.env.NODE_ENV === "development",
      runtime: "node",
      viewMode: "expanded",
      theme: moonlight,
    }),
  ],

  contextFieldName: "context",
  metadataFieldName: "metadata",
});
