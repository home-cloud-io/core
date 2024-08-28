import { createApi, fetchBaseQuery } from '@reduxjs/toolkit/query/react';
import { createConnectTransport } from '@connectrpc/connect-web';
import { createPromiseClient } from '@connectrpc/connect';
import { WebService } from 'api/platform/server/v1/web_connect';
import { setUserSettings } from './user_slice';

let BASE_URL = '';

if (process.env.NODE_ENV === 'production') {
  BASE_URL = 'http://home-cloud.local';
} else {
  BASE_URL = 'http://10.0.0.108:8000';
}

const web_service_transport = createConnectTransport({
  baseUrl: BASE_URL,
});

const client = createPromiseClient(WebService, web_service_transport);

export const serverRPCService = createApi({
  reducerPath: 'server_rpc_service',
  baseQuery: fetchBaseQuery({ baseUrl: BASE_URL }),
  endpoints: (builder) => ({
    // TODO: Update the main page to use these function instead of the direct calls
    shutdownHost: builder.mutation({
      queryFn: async () => {
        return client.shutdownHost({});
      },
    }),
    restartHost: builder.mutation({
      queryFn: async () => {
        return client.restartHost({});
      },
    }),
    installApp: builder.mutation({
      queryFn: async (req) => {
        return client.installApp(req);
      },
    }),
    deleteApp: builder.mutation({
      queryFn: async (req) => {
        return client.deleteApp(req);
      },
    }),
    updateApp: builder.mutation({
      queryFn: async (req) => {
        return client.updateApp(req);
      },
    }),
    // TODO: Add remaining endpoints here
    getIsDeviceSetup: builder.query({
      queryFn: async () => {
        const res = await client.isDeviceSetup({})
        return { data: { isDeviceSetup: res.setup }}
      },
    }),
    initDevice: builder.mutation({
      queryFn: async (req, store) => {
        let request = {
          username: store.getState().server.username,
          password: store.getState().server.password,
          timezone: store.getState().server.timezone,
        }

        if (store.getState().server.autoUpdateApps === "true") {
          request.auto_update_apps = true;
        } else {
          request.auto_update_apps = false;
        }

        if (store.getState().server.autoUpdateOs === "true") {
          request.auto_update_os = true;
        } else {
          request.auto_update_os = false;
        }

        const response = await client.initializeDevice(request);

        return { data: { isDeviceSetup: response.setup }};
      },
    }),
    login: builder.mutation({
      queryFn: async (req, store) => {
        
        const response = await client.login(req);

        store.dispatch(setUserSettings({ username: req.username, token: response.token }));
      
        return { data: { loggedIn: true }};
      }
    }),
    getAppStoreEntities: builder.query({
      queryFn: async () => {
        const res = await client.getAppsInStore({});

        return { data: res.data };
      },
    }),
  }),
});

export const { 
  useShutdownHostMutation,
  useRestartHostMutation,
  useInstallAppMutation,
  useDeleteAppMutation,
  useUpdateAppMutation,
  useGetIsDeviceSetupQuery,
  useInitDeviceMutation,
  useLoginMutation,
  useGetAppStoreEntitiesQuery
} = serverRPCService;

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
  });
}
