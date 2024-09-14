import * as React from "react";
import { Routes, Route } from "react-router-dom";

import Login from "../Login";
import DefaultLayout from "../DefaultLayout";
import DeviceOnboardPage from "../Device/Onboard";
import AppStorePage from "../AppStore";
import HomePage from "../Home";
import SettingsPage from "../Settings";

import {useGetIsDeviceSetupQuery} from "../../services/web_rpc";

export default function DashboardPage() {
  const { data, error, isLoading } = useGetIsDeviceSetupQuery();

  // TODO: make this way better
  if (isLoading) {
    return <div>Loading...</div>;
  }

  // TODO: make this way better
  if (error) {
    return <div>Error: {error.message}</div>;
  }

  // if the device is not setup, redirect to the onboarding page
  if (!isLoading && data.isDeviceSetup === false) {
    // TODO: refine this to use the react-router-dom redirect if possible
    window.history.pushState({}, '', '/getting-started');
    return <DeviceOnboardPage />;
  }

  return (
    <>
      <Routes>
        <Route path="/" element={<DefaultLayout />} >
          <Route path="home" element={<HomePage />} />
          <Route path="store" element={<AppStorePage />} />
          <Route path="settings" element={<SettingsPage />} />
        </Route>

        <Route path="getting-started" element={<DeviceOnboardPage/>} />
        <Route path="login" element={<Login/>} />
      </Routes>
    </>
  );
}
