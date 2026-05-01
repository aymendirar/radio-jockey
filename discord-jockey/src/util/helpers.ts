import { ConnectError } from "@connectrpc/connect";

export async function withConnectError<T, E = T>(
  fn: () => Promise<T>,
  onError: (err: ConnectError) => E | Promise<E>
): Promise<T | E> {
  try {
    return await fn();
  } catch (err) {
    if (err instanceof ConnectError) {
      return onError(err);
    }
    throw err;
  }
}
