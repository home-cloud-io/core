import { createSlice } from '@reduxjs/toolkit';

const initialState = {
  username: '',
  token: '',
};

export const userSlice = createSlice({
  name: 'user_settings',
  initialState,
  reducers: {
    setUserSettings: (state, action) => {
      state.username = action.payload.username;
      state.token = action.payload.token;
    },
  },
});

export const { setUserSettings } = userSlice.actions;

export default userSlice.reducer;
