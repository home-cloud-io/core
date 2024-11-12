import { createApi, fetchBaseQuery } from '@reduxjs/toolkit/query/react';
import { createConnectTransport } from '@connectrpc/connect-web';
import { createPromiseClient } from '@connectrpc/connect';
import { WebService } from 'api/platform/server/v1/web_connect';
import { setUserSettings } from './user_slice';
import * as Config from '../utils/config';

export const web_service_transport = createConnectTransport({
  baseUrl: Config.BASE_URL,
});

export const client = createPromiseClient(WebService, web_service_transport);

export const serverRPCService = createApi({
  reducerPath: 'server_rpc_service',
  baseQuery: fetchBaseQuery({ baseUrl: Config.BASE_URL }),
  endpoints: (builder) => ({
    getEvents: builder.query({
      queryFn: () => ({ data: [] }),
      async onCacheEntryAdded(
        arg,
        { updateCachedData, cacheDataLoaded, cacheEntryRemoved }
      ) {
        console.log('setting up events cache');
        try {
          // wait for the initial query to resolve before proceeding
          await cacheDataLoaded;

          const listen = async function () {
            try {
              // when data is received from the stream to the server,
              // if it is a message and for the appropriate channel,
              // update our query result with the received message
              for await (const event of client.subscribe({})) {
                // ignore heartbeats
                if (event.event.case === 'heartbeat') {
                  console.log('heartbeat');
                  continue;
                }

                const data = event.toJson();
                updateCachedData((draft) => {
                  draft.push(data);
                });
              }
            } catch (err) {
              console.warn('stream failed');
            }
          };
          listen();
          console.log('listening to event stream');
        } catch (err) {
          // no-op in case `cacheEntryRemoved` resolves before `cacheDataLoaded`,
          // in which case `cacheDataLoaded` will throw
          console.warn('subscription failed for cache: ', err);
        }
        // cacheEntryRemoved will resolve when the cache subscription is no longer active
        await cacheEntryRemoved;
        // perform cleanup steps once the `cacheEntryRemoved` promise resolves
        // client.close()
      },
    }),
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
      queryFn: async ({ app }) => {
        const req = {
          repo: 'home-cloud-io.github.io/store',
          chart: app.name,
          release: `${app.name}`,
          version: app.version,
        };

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
        try {
          const res = await client.updateApp(req);
          return { data: res.toJson() };
        } catch (error) {
          return { error: error.rawMessage };
        }
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
    enableSecureTunnelling: builder.mutation({
      queryFn: async (req) => {
        try {
          const res = await client.enableSecureTunnelling(req);
          return {};
        } catch (error) {
          return { error: error.rawMessage };
        }
      },
    }),
    disableSecureTunnelling: builder.mutation({
      queryFn: async (req) => {
        try {
          const res = await client.disableSecureTunnelling(req);
          return {};
        } catch (error) {
          return { error: error.rawMessage };
        }
      },
    }),
    registerToLocator: builder.mutation({
      queryFn: async (req) => {
        try {
          const res = await client.registerToLocator(req);
          return { data: res.toJson() };
        } catch (error) {
          return { error: error.rawMessage };
        }
      },
    }),
    deregisterFromLocator: builder.mutation({
      queryFn: async (req) => {
        try {
          const res = await client.deregisterFromLocator(req);
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
          const resInstalled = await client.appsHealthCheck({});
          const resStore = await client.getAppsInStore({});

          const apps = [];
          for (const storeApp of resStore.apps) {
            let app = {
              name: storeApp.name,
              version: storeApp.version,
              icon: storeApp.icon,
              digest: storeApp.digest,
              readme: storeApp.readme,
              installed: false,
            };
            for (const installedApp of resInstalled.checks) {
              if (storeApp.name === installedApp.name) {
                app.installed = true;
                break;
              }
            }
            apps.push(app);
          }

          return { data: apps };
        } catch (error) {
          return { error: error.rawMessage };
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
    getAppStorage: builder.query({
      queryFn: async () => {
        try {
          const res = await client.getAppStorage({});
          return { data: res.toJson().apps };
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
  useEnableSecureTunnellingMutation,
  useDisableSecureTunnellingMutation,
  useRegisterToLocatorMutation,
  useDeregisterFromLocatorMutation,
  useLoginMutation,
  useGetAppStoreEntitiesQuery,
  useGetAppsHealthCheckQuery,
  useGetDeviceSettingsQuery,
  useGetSystemStatsQuery,
  useGetAppStorageQuery,
  useGetEventsQuery,
} = serverRPCService;
