import { configureStore } from '@reduxjs/toolkit';
import { setupListeners } from '@reduxjs/toolkit/query';

import { serverRPCService } from '../services/web_rpc';

import {
  streamingClient,
  subscribeMiddleware,
} from '../services/event_stream';

import serverSlice from '../services/web_slice';
import userSlice from '../services/user_slice';

export const store = configureStore({
  reducer: {
    server: serverSlice,
    user_settings: userSlice,
    [serverRPCService.reducerPath]: serverRPCService.reducer,
  },
  middleware: (getDefaultMiddleware) =>
    getDefaultMiddleware().concat([
      serverRPCService.middleware,
      subscribeMiddleware(streamingClient),
    ]),
})

setupListeners(store.dispatch)