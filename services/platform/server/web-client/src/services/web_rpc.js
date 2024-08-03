import { createConnectTransport } from '@connectrpc/connect-web';
import { createPromiseClient } from '@connectrpc/connect';
import { WebService } from 'api/platform/server/v1/web_connect';

const transport = createConnectTransport({
  baseUrl: 'http://home-cloud.local',
});

const client = createPromiseClient(WebService, transport);

export function shutdown() {
  console.log("shutdown called")
  client.shutdownHost({})
}

export function restart() {
  console.log("restart called")
  client.restartHost({})
}

