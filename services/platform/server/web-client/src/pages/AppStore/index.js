import * as React from 'react';
import { useState } from 'react';
import ProgressBar from 'react-bootstrap/ProgressBar';
import Button from 'react-bootstrap/Button';
import { marked } from 'marked';
import {
  AppInstallStatus,
  setAppInstallStatus,
} from '../../services/web_slice';
import {
  useGetEventsQuery,
  useGetAppStoreEntitiesQuery,
  useInstallAppMutation,
  useDeleteAppMutation,
} from '../../services/web_rpc';

export default function AppStorePage() {
  const { data, error, isLoading } = useGetAppStoreEntitiesQuery();
  const { data: events } = useGetEventsQuery();

  const ListEntries = () => {
    return (
      <div className="my-3 p-3 bg-body rounded shadow-sm">
        <h4 className="header border-bottom">
          <b>App Store</b>
        </h4>
        {data.map((app) => {
          return <StoreEntry events={events} app={app} key={app.digest} />;
        })}
      </div>
    );
  };

  return (
    <>
      {isLoading ? (
        <p>Loading...</p>
      ) : error ? (
        <p>Error: {error}</p>
      ) : (
        <ListEntries />
      )}
    </>
  );
}

function StoreEntry({ events = [], app }) {
  const [installApp] = useInstallAppMutation();
  const [deleteApp] = useDeleteAppMutation();
  const [status, setStatus] = useState(AppInstallStatus.DEFAULT);

  if (status === AppInstallStatus.DEFAULT) {
    setStatus(
      app.installed ? AppInstallStatus.INSTALLED : AppInstallStatus.UNINSTALLED
    );
  }

  if (events.length > 0) {
    const latestEvent = events.at(-1)['appInstalled'];
    if (latestEvent && latestEvent.name === app.name) {
      if (status != AppInstallStatus.INSTALLED) {
        setStatus(AppInstallStatus.INSTALLED);
      }
    }
  }

  const handleAppInstallClick = (app) => {
    setStatus(AppInstallStatus.INSTALLING);
    installApp({ app });
  };

  const handleAppUninstallClick = (app) => {
    // TODO: recieve an event back from the server when uninstalling is done
    setStatus(AppInstallStatus.UNINSTALLED);
    deleteApp(app.name);
  };

  const rowStyles = {
    paddingLeft: '2rem',
  };

  const btnStyles = {
    marginTop: '-2.5rem',
  };

  return (
    <div className="d-flex text-body-secondary pt-3">
      <div>
        <img src={app.icon} width={48} height={48} />
      </div>

      <div
        className="pb-3 mb-0 small lh-sm border-bottom w-100 position-relative"
        style={rowStyles}
      >
        <div>
          Version: <em>{app.version}</em>
        </div>
        <div dangerouslySetInnerHTML={{ __html: marked.parse(app.readme) }} />

        <div>
          {status === AppInstallStatus.UNINSTALLED && (
            <Button
              variant="success"
              className="float-end"
              onClick={() => handleAppInstallClick(app)}
            >
              Install
            </Button>
          )}
          {status === AppInstallStatus.INSTALLED && (
            <Button
              variant="secondary"
              className="float-end"
              onClick={() => handleAppUninstallClick(app)}
            >
              Uninstall
            </Button>
          )}
        </div>

        <div>
          {status === AppInstallStatus.INSTALLING && (
            <ProgressBar animated now={100} />
          )}
        </div>
      </div>
    </div>
  );
}
