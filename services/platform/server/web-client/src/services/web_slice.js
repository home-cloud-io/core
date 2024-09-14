import { createSlice } from '@reduxjs/toolkit';

export const EventConnectionStatus = Object.freeze({
    CONNECTED: 'connected',
    CONNECTING: 'connecting',
    DISCONNECTED: 'disconnected',
    ERROR: 'error',
});

const initialState = {
    username: '',
    event_stream_connection_status: EventConnectionStatus.DISCONNECTED,
    event: [],
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

            state.event.push(action.payload.data);
        },
    }
});

export const {
    setUser,
    setDeviceSettings,
    setEventStreamConnectionStatus,
    setEvent,
} = serverSlice.actions;

export default serverSlice.reducer;