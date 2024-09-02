import * as React from "react";
import { Routes, Route } from "react-router-dom";
import { useSelector } from "react-redux";

import Login from "../Login";
import DefaultLayout from "../DefaultLayout";
import DeviceOnboardPage from "../Device/Onboard";

import AppStorePage from "../AppStore";
import HomePage from "../Home";
import SettingsPage from "../Settings";

import {useGetIsDeviceSetupQuery} from "../../services/web_rpc";

export default function DashboardPage() {
  const userSettings = useSelector((state) => state.user_settings);
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

  if (!isLoading && data.isDeviceSetup === true) {
    window.history.pushState({}, '', '/home');
    return <HomePage />;
  }

  // if the device is setup and the user is not logged in redirect to the login page
  // TODO: Figure out why state keeps getting reset on page reload.
  //       Most likely need to use local storage or cookies to persist the token anyways
  // if (!isLoading && data.isDeviceSetup === true && userSettings.token === "") {
  //   window.history.pushState({}, '', '/login');
  //   return <Login />;
  // }

  return (
    <>
      <Routes>
        <Route path="/" element={<DefaultLayout />} >
          <Route index path="home" element={<HomePage />} />
          <Route path="store" element={<AppStorePage />} />
          <Route path="settings" element={<SettingsPage />} />
        </Route>

        <Route path="getting-started" element={<DeviceOnboardPage/>} />
        <Route path="login" element={<Login/>} />
      </Routes>
    </>
  );
}
