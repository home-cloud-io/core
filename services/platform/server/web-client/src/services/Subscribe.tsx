import React, {
  ReactNode,
  createContext,
  useContext,
  useEffect,
  useState,
} from 'react';
import { createConnectTransport } from '@connectrpc/connect-web';
import { createClient } from '@connectrpc/connect';
import { WebService } from 'api/platform/server/v1/web_connect';
import { ServerEvent } from 'api/platform/server/v1/web_pb';
import * as Config from '../utils/config';

export const SubscribeContext = React.createContext({ client: WebService });

export const transport = createConnectTransport({
  baseUrl: Config.BASE_URL,
});

export const client = createClient(WebService, transport);

export type ProviderValue = {
  connected: boolean;
  setConnected: React.Dispatch<React.SetStateAction<boolean>>;
  event: ServerEvent | undefined;
  setEvent: React.Dispatch<React.SetStateAction<ServerEvent | undefined>>;
};
type DefaultValue = undefined;
type ContextValue = DefaultValue | ProviderValue;

const EventContext = createContext<ContextValue>(undefined);

export function useEvents() {
  return useContext(EventContext);
}

export type Props = {
  children: ReactNode;
};

export function EventsProvider(props: Props) {
  const { children } = props;

  const [connected, setConnected] = useState(false);
  const [event, setEvent] = useState<ServerEvent>();
  const value = {
    connected,
    setConnected,
    event,
    setEvent,
  };

  return (
    <EventContext.Provider value={value}>{children}</EventContext.Provider>
  );
}

// NOTE: this will load twice because of React.StrictMode loading all components twice
export function EventListener() {
  const { setConnected, setEvent } = useEvents() as ProviderValue;

  useEffect(() => {
    console.log('initializing event listener');
    const listen = async function () {
      try {
        for await (const event of client.subscribe({})) {
          setConnected(true);
          // ignore heartbeats
          if (event.event.case === 'heartbeat') {
            continue;
          }
          setEvent(event);
        }
      } catch (err) {
        console.warn('stream failed');
        setConnected(false);
        await new Promise((f) => setTimeout(f, 1000));
      }
    };
    (async () => {
      while (true) {
        console.log('connecting to event stream');
        await listen();
      }
    })();
  });

  return <></>;
}
