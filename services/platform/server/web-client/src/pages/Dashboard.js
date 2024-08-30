import * as React from "react";
import { Routes, Route } from "react-router-dom";
import { Routes, Route, redirect } from "react-router-dom";
import { useSelector } from "react-redux";

import Login from "./Login";
import DefaultLayout from "./DefaultLayout";
import DeviceOnboardPage from "./Device/Onboard";
import AppStore from "./AppStore/AppStore";

import {useGetIsDeviceSetupQuery} from "../services/web_rpc";

import { HomePage } from "./Home";

export default function Dashboard() {
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
        <Route index element={<Home/>} />
        <Route path="/" element={<DefaultLayout />} >
          <Route path="home" element={<HomePage />} />
          <Route path="store" element={<AppStore />} />
        </Route>

        <Route path="getting-started" element={<DeviceOnboardPage/>} />
        <Route path="login" element={<Login/>} />
      </Routes>
    </>
  );
}
