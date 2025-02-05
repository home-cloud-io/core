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
import { Log } from 'api/platform/server/v1/web_pb';
import * as Config from '../utils/config';

export const SubscribeContext = React.createContext({ client: WebService });

export const transport = createConnectTransport({
  baseUrl: Config.BASE_URL,
});

export const client = createClient(WebService, transport);

export type ProviderValue = {
  connected: boolean;
  setConnected: React.Dispatch<React.SetStateAction<boolean>>;
  log: Log | undefined;
  setLog: React.Dispatch<React.SetStateAction<Log | undefined>>;
};
type DefaultValue = undefined;
type ContextValue = DefaultValue | ProviderValue;

const LogContext = createContext<ContextValue>(undefined);

export function useLogs() {
  return useContext(LogContext);
}

export type Props = {
  children: ReactNode;
};

export function LogProvider(props: Props) {
  const { children } = props;

  const [connected, setConnected] = useState(false);
  const [log, setLog] = useState<Log>();
  const value = {
    connected,
    setConnected,
    log,
    setLog,
  };

  return (
    <LogContext.Provider value={value}>{children}</LogContext.Provider>
  );
}

// NOTE: this will load twice because of React.StrictMode loading all components twice
export function LogListener() {
  const { setConnected, setLog } = useLogs() as ProviderValue;

  useEffect(() => {
    console.log('initializing log listener');
    const listen = async function () {
      try {
        for await (const log of client.logs({})) {
          setConnected(true);
          setLog(log);
        }
      } catch (err) {
        console.warn(`log stream failed: ${err}`);
        setConnected(false);
        await new Promise((f) => setTimeout(f, 1000));
      }
    };
    (async () => {
      while (true) {
        console.log('connecting to log stream');
        await listen();
      }
    })();
  });

  return <></>;
}
