import * as React from "react";
import { Routes, Route, redirect } from "react-router-dom";

import Login from "./Login";
import Home from "./Home";
import DeviceOnboardPage from "./Device/Onboard";
import {useGetIsDeviceSetupQuery} from "../services/web_rpc";

export default function Dashboard() {
  // TODO: Check BE for device status and if not onboarded, redirect to onboard page
  //       this will basically be setting the pattern for how the rest of the application
  //       will handle state, and rpc calls.
  const { data, error, isLoading } = useGetIsDeviceSetupQuery();

  const clickTest = () => {
    console.log(data);
  }

  if (isLoading) {
    return <div>Loading...</div>;
  } 

  if (!isLoading && data.isDeviceSetup === false) {
    // TODO: refine this to use the react-router-dom redirect if possible
    window.history.pushState({}, '', '/getting-started');
    return <DeviceOnboardPage />; 
  }
 
  return (
    <>
      <button onClick={clickTest}>Click me</button>
      <Routes>
        <Route index element={<Home/>} />
        <Route path="getting-started" element={<DeviceOnboardPage/>} />
        <Route path="login" element={<Login/>} />
      </Routes>
    </>
  );
}
