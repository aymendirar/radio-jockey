import { sign } from "paseto-ts/v4";
import { Code } from "@connectrpc/connect";
import { radioClient } from "../client.js";
import { getAuthToken, setAuthToken } from "../../discord/sessions.js";
import { withConnectError } from "../../util/helpers.js";

const privateKey = process.env.PRIVATE_PASETO_KEY!;

function signNonce(nonce: string): string {
  return sign(privateKey, { nonce });
}

async function acquireAuthToken(sessionId: string): Promise<string> {
  const { nonce } = await radioClient.requestNonce({});
  const { authToken } = await radioClient.respondNonce({ passKey: signNonce(nonce) });
  setAuthToken(sessionId, authToken);
  return authToken;
}

async function withAuth<T>(sessionId: string, fn: (authToken: string) => Promise<T>): Promise<T> {
  const cached = getAuthToken(sessionId);
  if (cached) {
    return withConnectError(
      () => fn(cached),
      async (err) => {
        if (err.code !== Code.Unauthenticated) throw err;
        const authToken = await acquireAuthToken(sessionId);
        return fn(authToken);
      },
    );
  }
  const authToken = await acquireAuthToken(sessionId);
  return fn(authToken);
}

export async function deleteSession(sessionId: string): Promise<void> {
  await withAuth(sessionId, (authToken) =>
    radioClient.deleteSessionAuth({ sessionId }, { headers: { authorization: `Bearer ${authToken}` } }),
  );
}
