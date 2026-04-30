import { ConnectError } from "@connectrpc/connect";

export async function withConnectError<T>(
  fn: () => Promise<T>,
  onError: (err: ConnectError) => T | Promise<T>
): Promise<T> {
  try {
    return await fn();
  } catch (err) {
    if (err instanceof ConnectError) {
      return onError(err);
    }
    throw err;
  }
}
