import { createApi, fetchBaseQuery } from '@reduxjs/toolkit/query/react';
import { createCallbackClient } from '@connectrpc/connect';
import { WebService } from 'api/platform/server/v1/web_connect';
import { client, web_service_transport } from './web_rpc';

import {
    setEvent,
    setEventStreamConnectionStatus,
    EventConnectionStatus
} from './web_slice';

let BASE_URL = '';
let LOCAL_DOMAIN = 'localhost';

if (process.env.NODE_ENV === 'development') {
  BASE_URL = `http://${LOCAL_DOMAIN}:8000`;
} else {
  BASE_URL = 'http://home-cloud.local';
}

const delay = ms => new Promise(res => setTimeout(res, ms));

export const streamingClient = createCallbackClient(WebService, web_service_transport);
export const SUBSCRIBE_EVENTS_ACTION = 'events/subscribe';

export const subscribeMiddleware = (client) => (params) => (next) => async (action) => {
  const { dispatch, getState } = params;

  if (action.type === SUBSCRIBE_EVENTS_ACTION) {

    if (getState().server.event_stream_connection_status === EventConnectionStatus.CONNECTED) {
      return next(action);
    }

    if (getState().server.event_stream_connection_status === EventConnectionStatus.CONNECTING) {
      return next(action);
    }

    // check if the store is already subscribed
    dispatch(setEventStreamConnectionStatus({ status: EventConnectionStatus.CONNECTING }));

    let done = false;
    for (let i = 0; i < 5; i++) {
      client.subscribe({}, (res) => {
          dispatch(setEvent({ data: res.toJson() }));
          done = true;
      }, (err) => {
          if (err) {
              console.warn("Error subscribing to events: ", err);
              dispatch(setEventStreamConnectionStatus({ status: EventConnectionStatus.ERROR }));

          }
      });
      if (done) {
        break;
      }
      console.log("retrying event stream")
      await delay(1000);
    }

    dispatch(setEventStreamConnectionStatus({ status: EventConnectionStatus.CONNECTED }));
  }

  return next(action);
}