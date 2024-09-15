import { createApi, fetchBaseQuery } from '@reduxjs/toolkit/query/react';
import { createConnectTransport } from '@connectrpc/connect-web';
import { createPromiseClient } from '@connectrpc/connect';
import { WebService } from 'api/platform/server/v1/web_connect';
import { setUserSettings } from './user_slice';

let BASE_URL = '';
let LOCAL_DOMAIN = 'localhost';

if (process.env.NODE_ENV === 'development') {
  BASE_URL = `http://${LOCAL_DOMAIN}:8000`;
} else {
  BASE_URL = 'http://home-cloud.local';
}

export const web_service_transport = createConnectTransport({
  baseUrl: BASE_URL,
});

export const client = createPromiseClient(WebService, web_service_transport);

export const serverRPCService = createApi({
  reducerPath: 'server_rpc_service',
  baseQuery: fetchBaseQuery({ baseUrl: BASE_URL }),
  endpoints: (builder) => ({
    shutdownHost: builder.mutation({
      queryFn: async () => {
        try {
          const res = await client.shutdownHost({});
          return { data: res.toJson() };
        } catch (error) {
          return { error: error.rawMessage };
        }
      },
    }),
    restartHost: builder.mutation({
      queryFn: async () => {
        try {
          const res = await client.restartHost({});
          return { data: res.toJson() };
        } catch (error) {
          return { error: error.rawMessage };
        }
      },
    }),
    installApp: builder.mutation({
      queryFn: async ({app}) => {
        const req = {
            repo: 'home-cloud-io.github.io/store',
            chart: app.name,
            release: `${app.name}`,
            version: app.version,
        }

        try {
          const res = await client.installApp(req);
          return { data: res.toJson() };
        } catch (error) {
          return { error: error.rawMessage };
        }
      },
    }),
    deleteApp: builder.mutation({
      queryFn: async (name) => {
        try {
          const res = await client.deleteApp({
            release: name,
          });
          return { data: res.toJson() };
        } catch (error) {
          return { error: error.rawMessage };
        }
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
          return { data: { isDeviceSetup: res.setup } };
        } catch (error) {
          return { error: error.rawMessage };
        }
      },
    }),
    initDevice: builder.mutation({
      queryFn: async (req) => {
        try {
          const res = await client.initializeDevice(req);
          return { data: { isDeviceSetup: res.toJson().setup } };
        } catch (error) {
          return { error: error.rawMessage };
        }
      },
    }),
    setDeviceSettings: builder.mutation({
      queryFn: async (req) => {
        try {
          const res = await client.setDeviceSettings(req);
          return {};
        } catch (error) {
          return { error: error.rawMessage };
        }
      },
    }),
    login: builder.mutation({
      queryFn: async (req, store) => {
        try {
          const res = await client.login(req);
          store.dispatch(
            setUserSettings({ username: req.username, token: res.token })
          );
          return { data: { user: res.toJson() } };
        } catch (error) {
          return { error: error.rawMessage };
        }
      },
    }),
    getAppStoreEntities: builder.query({
      queryFn: async () => {
        try {
          const res = await client.getAppsInStore({});
          return { data: res.toJson().apps };
        } catch (error) {
          return { error };
        }
      },
    }),
    getAppsHealthCheck: builder.query({
      queryFn: async () => {
        try {
          const res = await client.appsHealthCheck({});
          return { data: res.toJson() };
        } catch (error) {
          return { error: error.rawMessage };
        }
      },
    }),
    getDeviceSettings: builder.query({
      queryFn: async () => {
        try {
          const res = await client.getDeviceSettings({});
          return { data: res.toJson().settings };
        } catch (error) {
          return { error: error.rawMessage };
        }
      },
    }),
    getSystemStats: builder.query({
      queryFn: async () => {
        try {
          const res = await client.getSystemStats({});
          return { data: res.toJson().stats };
        } catch (error) {
          return { error: error.rawMessage };
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
  useSetDeviceSettingsMutation,
  useLoginMutation,
  useGetAppStoreEntitiesQuery,
  useGetAppsHealthCheckQuery,
  useGetDeviceSettingsQuery,
  useGetSystemStatsQuery,
} = serverRPCService;
