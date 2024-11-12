import { createSlice } from '@reduxjs/toolkit';

export const EventConnectionStatus = Object.freeze({
  CONNECTED: 'connected',
  CONNECTING: 'connecting',
  DISCONNECTED: 'disconnected',
  ERROR: 'error',
});

export const AppInstallStatus = Object.freeze({
  DEFAULT: 'default',
  INSTALLING: 'installing',
  INSTALLED: 'installed',
  UNINSTALLING: 'uninstalling',
  UNINSTALLED: 'uninstalled',
  ERROR: 'error',
});

export const FileUploadStatus = Object.freeze({
  DEFAULT: 'default',
  UPLOADING: 'uploading',
  COMPLETE: 'complete',
  ERROR: 'error',
});

const initialState = {
  username: '',
  event_stream_connection_status: EventConnectionStatus.DISCONNECTED,
  event: [],
  app_install_status: {},
};

export const serverSlice = createSlice({
  name: 'server',
  initialState,
  reducers: {
    setUser: (state, action) => {
      state.username = action.payload.username;
      state.password = action.payload.password;
    },
    setDeviceSettings: (state, action) => {
      const { timezone, autoUpdateApps, autoUpdateOs } = action.payload;
      state.timezone = timezone;
      state.autoUpdateApps = autoUpdateApps;
      state.autoUpdateOs = autoUpdateOs;
    },
    setEventStreamConnectionStatus: (state, action) => {
      state.event_stream_connection_status = action.payload.status;
    },
  },
});

export const { setUser, setDeviceSettings, setEventStreamConnectionStatus } =
  serverSlice.actions;

export default serverSlice.reducer;
