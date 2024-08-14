import { configureStore } from '@reduxjs/toolkit';
import { setupListeners } from '@reduxjs/toolkit/query';

import { keyValueRPCService } from '../services/key_value_rpc';
import { serverRPCService } from '../services/web_rpc';

export const store = configureStore({
  reducer: {
    [serverRPCService.reducerPath]: serverRPCService.reducer,
    [keyValueRPCService.reducerPath]: keyValueRPCService.reducer,
  },
  middleware: (getDefaultMiddleware) => 
    getDefaultMiddleware().concat([keyValueRPCService.middleware, serverRPCService.middleware]), 
})

setupListeners(store.dispatch)