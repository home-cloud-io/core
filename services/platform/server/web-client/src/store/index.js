import { configureStore } from '@reduxjs/toolkit';
import { setupListeners } from '@reduxjs/toolkit/query';

import { keyValueRPCService } from '../services/key_value_rpc';

export const store = configureStore({
  reducer: {
    [keyValueRPCService.reducerPath]: keyValueRPCService.reducer,
  },
  middleware: (getDefaultMiddleware) => 
    getDefaultMiddleware().concat(keyValueRPCService.middleware), 
})

setupListeners(store.dispatch)