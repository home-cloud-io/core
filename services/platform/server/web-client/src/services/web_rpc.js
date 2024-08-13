import { createConnectTransport } from '@connectrpc/connect-web';
import { createPromiseClient } from '@connectrpc/connect';
import { WebService } from 'api/platform/server/v1/web_connect';

const transport = createConnectTransport({
  baseUrl: 'http://home-cloud.local',
  // baseUrl: 'http://localhost:8000',
});

const values = new Map([
  [
    'hello-world',
    ``,
  ],
  [
    'postgres',
    ``,
  ],
  [
    'immich',
    ``,
  ],
]);

const versions = new Map([
  [
    'hello-world',
    '0.0.5',
  ],
  [
    'postgres',
    '0.0.7',
  ],
  [
    'immich',
    '0.0.12',
  ],
]);



const client = createPromiseClient(WebService, transport);

export function shutdown() {
  console.log('shutdown called');
  client.shutdownHost({});
}

export function restart() {
  console.log('restart called');
  client.restartHost({});
}

export function installApp(app) {
  console.log(`installing app: ${app}`);
  client.installApp({
    repo: 'home-cloud-io.github.io/store',
    chart: app,
    release: `${app}`,
    values: values.get(app),
    version: versions.get(app)
  });
}

export function deleteApp(app) {
  console.log('delete app called');
  client.deleteApp({
    release: `${app}`,
  });
}

export function updateApp(app) {
  console.log('update app called');
  client.updateApp({
    repo: 'home-cloud-io.github.io/store',
    chart: app,
    release: `${app}`,
    values: values.get(app),
    version: versions.get(app)
  });
}
