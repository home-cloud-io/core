import React from 'react';
import ReactDOM from 'react-dom/client';
import { BrowserRouter as Router } from 'react-router-dom';
import './index.css';
import Entrypoint from './App';
import { createConnectTransport } from '@connectrpc/connect-web';
import { TransportProvider } from '@connectrpc/connect-query';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { EventListener, EventsProvider } from './services/Subscribe';
import * as Config from './utils/config';

export const transport = createConnectTransport({
  baseUrl: Config.BASE_URL,
});

const queryClient = new QueryClient();

const root = ReactDOM.createRoot(
  document.getElementById('root') as HTMLElement
);
root.render(
  <React.StrictMode>
    <TransportProvider transport={transport}>
      <EventsProvider>
        <EventListener />
        <QueryClientProvider client={queryClient}>
          <Router>
            <Entrypoint />
          </Router>
        </QueryClientProvider>
      </EventsProvider>
    </TransportProvider>
  </React.StrictMode>
);
