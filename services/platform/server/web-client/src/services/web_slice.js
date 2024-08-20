import { createSlice } from '@reduxjs/toolkit';

// TODO: Update the initial state to match the server request. Investigate if the empty request can be the initial state
// of the slice.
const initialState = {
    username: '',
    password: '',
    timezone: '',
    autoUpdateApps: '',
    autoUpdateOs: '', 
    default_apps: [],
}
  
export const serverSlice = createSlice({
    name: 'device_setup',
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
        setDefaultApps: (state, action) => {
            state.default_apps = action.payload;
        }
    }
});
  
export const {
    setUser,
    setDeviceSettings,
    setDefaultApps
} = serverSlice.actions;
  
  export default serverSlice.reducer;