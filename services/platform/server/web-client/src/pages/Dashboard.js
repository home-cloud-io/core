import * as React from "react";
import { Routes, Route } from "react-router-dom";

import Login from "./Login";
import Home from "./Home";

export default function Dashboard() {
  return (
    <>
      <Routes>
        <Route index element={<Home/>} />
        <Route path="login" element={<Login/>} />
      </Routes>
    </>
  );
}
