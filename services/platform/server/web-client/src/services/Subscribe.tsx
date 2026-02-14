import React, {
  ReactNode,
  createContext,
  useContext,
  useEffect,
  useState,
} from "react";
import { createConnectTransport } from "@connectrpc/connect-web";
import { createClient } from "@connectrpc/connect";
import { WebService } from "@home-cloud/api/platform/server/v1/web_pb";
import { ServerEvent } from "@home-cloud/api/platform/server/v1/web_pb";

export const SubscribeContext = React.createContext({ client: WebService });

export const transport = createConnectTransport({
  baseUrl: window.location.origin,
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

// NOTE: this doesn't work when running in npm dev mode (I think because the npm proxy can't handle the http2 stream?)
export function EventListener() {
  const { setConnected, setEvent } = useEvents() as ProviderValue;

  useEffect(() => {
    console.log("initializing event listener");
    const listen = async function () {
      try {
        console.log("listening for events");
        for await (const res of client.subscribe({})) {
          console.log("received event");
          setConnected(true);
          // ignore heartbeats
          if (res.event.case === "heartbeat") {
            continue;
          }
          setEvent(res);
        }
      } catch (err) {
        console.warn(`event stream failed: ${err}`);
        setConnected(false);
        await new Promise((f) => setTimeout(f, 1000));
      }
    };
    (async () => {
      while (true) {
        console.log("connecting to event stream");
        await listen();
        console.log("disconnected event stream");
      }
    })();
  });

  return <></>;
}
