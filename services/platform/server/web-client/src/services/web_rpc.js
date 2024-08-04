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
    `
replicaCount: 1
`,
  ],
  [
    'postgres',
    `
nodeAffinity:
  hostname: home-cloud
`,
  ],
  [
    'immich',
    `
database:
  name: postgres
  user: postgres
  password: postgres
nodeAffinity:
  hostname: home-cloud
`,
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
    repo: 'jack.kawell.us/helm-charts',
    chart: app,
    release: `home-cloud-${app}`,
    values: values.get(app),
  });
}

export function deleteApp(app) {
  console.log('delete app called');
  client.deleteApp({
    release: `home-cloud-${app}`,
  });
}

export function updateApp(app) {
  console.log('update app called');
  client.updateApp({
    repo: 'jack.kawell.us/helm-charts',
    chart: app,
    release: `home-cloud-${app}`,
    values: values.get(app),
  });
}
