import React from "react";
import { createRoot } from "react-dom/client";
import { BrowserRouter } from "react-router-dom";
import Dashboard from "./pages/Dashboard";

import '../public/globals.css';

const rootElement = document.getElementById("root");
const root = createRoot(rootElement);

root.render(
  <React.StrictMode>
      <BrowserRouter>
        <Dashboard />
      </BrowserRouter>
  </React.StrictMode>
);
