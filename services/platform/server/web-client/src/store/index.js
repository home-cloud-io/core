import { configureStore } from '@reduxjs/toolkit';
import { setupListeners } from '@reduxjs/toolkit/query';

import { keyValueRPCService } from '../services/key_value_rpc';
import { serverRPCService } from '../services/web_rpc';
import serverSlice from '../services/web_slice';
import userSlice from '../services/user_slice';

export const store = configureStore({
  reducer: {
    server: serverSlice,
    user_settings: userSlice,
    [serverRPCService.reducerPath]: serverRPCService.reducer,
    [keyValueRPCService.reducerPath]: keyValueRPCService.reducer,
  },
  middleware: (getDefaultMiddleware) => 
    getDefaultMiddleware().concat([keyValueRPCService.middleware, serverRPCService.middleware]), 
})

setupListeners(store.dispatch)