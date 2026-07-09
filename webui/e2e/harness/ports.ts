import * as net from 'node:net';

/**
 * Allocates a free TCP port by briefly listening on port 0 and closing the
 * listener. There is an inherent race between releasing the port and the
 * backrest process binding it, so callers should retry spawn on bind failure
 * (BackrestInstance.start does this).
 */
export async function getFreePort(): Promise<number> {
  return new Promise((resolve, reject) => {
    const server = net.createServer();
    server.unref();
    server.on('error', reject);
    server.listen(0, '127.0.0.1', () => {
      const address = server.address();
      if (address === null || typeof address === 'string') {
        server.close(() => reject(new Error('failed to allocate a port')));
        return;
      }
      const port = address.port;
      server.close((err) => (err ? reject(err) : resolve(port)));
    });
  });
}
