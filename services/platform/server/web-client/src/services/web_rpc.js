import { createApi, fetchBaseQuery } from '@reduxjs/toolkit/query/react';
import { createConnectTransport } from '@connectrpc/connect-web';
import { createPromiseClient } from '@connectrpc/connect';
import { WebService } from 'api/platform/server/v1/web_connect';
import { setUserSettings } from './user_slice';

let BASE_URL = '';

if (process.env.NODE_ENV === 'production') {
  BASE_URL = 'http://home-cloud.local';
} else {
  BASE_URL = 'http://localhost:8000';
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
      queryFn: async (app) => {
        try {
          const res = await client.installApp({
            repo: 'home-cloud-io.github.io/store',
            chart: app,
            release: `${app}`,
            values: values.get(app),
          });

          return { data: res };
        } catch (error) {
          console.log(error);
          return { error };
        }
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
        try {
          const res = await client.isDeviceSetup({});
          return { data: { isDeviceSetup: res.setup }};
        } catch (error) {
          return { error: error.rawMessage };
        }
      },
    }),
    initDevice: builder.mutation({
      queryFn: async (req) => {
        try {
          const res = await client.initializeDevice(req);
          return { data: { isDeviceSetup: res.setup }};
        } catch (error) {
          console.log(error);
          return { error: error.rawMessage };
        }
      },
    }),
    login: builder.mutation({
      queryFn: async (req, store) => {
        try {
          const response = await client.login(req);
          // dispatch a redux action to set the user settings
          store.dispatch(setUserSettings({ username: req.username, token: response.token }));
          return { data: { loggedIn: true }};
        } catch (error) {
          return { error: error.rawMessage };
        }  
      }
    }),
    getAppStoreEntities: builder.query({
      queryFn: async () => {
        try {
          const res = await client.getAppsInStore({});
          return { data: res.toJson().apps};
        } catch (error) {
          return { error };
        }
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
  useGetAppStoreEntitiesQuery,
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