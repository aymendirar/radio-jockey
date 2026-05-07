import { PassThrough } from "node:stream";
import { VoiceConnection } from "@discordjs/voice";

type Session = {
  connection: VoiceConnection;
  buffer: PassThrough | null;
  authToken: string | null;
};

const sessions = new Map<string, Session>();

export function hasSession(guildId: string): boolean {
  return sessions.has(guildId);
}

export function getAuthToken(guildId: string): string | null {
  return sessions.get(guildId)?.authToken ?? null;
}

export function setAuthToken(guildId: string, authToken: string) {
  const session = sessions.get(guildId);
  if (session) session.authToken = authToken;
}

export function createSession(guildId: string, connection: VoiceConnection) {
  sessions.set(guildId, { connection, buffer: null, authToken: null });
}

export function setBuffer(guildId: string, buffer: PassThrough) {
  const session = sessions.get(guildId);
  if (session) session.buffer = buffer;
}

export function destroySession(guildId: string) {
  sessions.get(guildId)?.connection.destroy();
  sessions.delete(guildId);
}
