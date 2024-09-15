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
    ERROR: 'error',
});

const initialState = {
    username: '',
    event_stream_connection_status: EventConnectionStatus.DISCONNECTED,
    event: [],
    app_install_status: {},
}

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
        setEvent: (state, action) => {
            // filter out heartbeat events
            if (action.payload.data.heartbeat) {
                return;
            }

            if (action.payload.data.error) {
                console.error("Received error event: ", action.payload.data.error);
                return;
            }

            if (action.payload.data.appInstalled) {
                state.app_install_status[action.payload.data.appInstalled.name] = AppInstallStatus.INSTALLED;
                return;
            }

            state.event.push(action.payload.data);
        },
        setAppInstallStatus: (state, action) => {
            const { app, status } = action.payload;
            state.app_install_status[app.name] = status;
        },
    }
});

export const {
    setAppInstallStatus,
    setUser,
    setDeviceSettings,
    setEventStreamConnectionStatus,
    setEvent,
} = serverSlice.actions;

export default serverSlice.reducer;