import React from 'react';
import { createRoot } from 'react-dom/client';
import { BrowserRouter } from 'react-router-dom';
import { Provider } from 'react-redux';

import DashboardPage from './pages/Dashboard';

// client side application state
import { store } from './store';

import '../public/globals.css';

const rootElement = document.getElementById('root');
const root = createRoot(rootElement);

root.render(
  <React.StrictMode>
    <Provider store={store}>
      <BrowserRouter>
        <DashboardPage />
      </BrowserRouter>
    </Provider>
  </React.StrictMode>
);
