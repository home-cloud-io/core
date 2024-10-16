import * as React from 'react';
import { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import ProgressBar from 'react-bootstrap/ProgressBar';
import { marked } from 'marked';
import { AppInstallStatus, setAppInstallStatus } from '../../services/web_slice';
import { useGetEventsQuery, useGetAppStoreEntitiesQuery, useInstallAppMutation, useDeleteAppMutation } from '../../services/web_rpc';

export default function AppStorePage() {
  const { data, error, isLoading } = useGetAppStoreEntitiesQuery();
  const { data: events } = useGetEventsQuery();

  const ListEntries = () => {
    return (
      <div className="my-3 p-3 bg-body rounded shadow-sm">
        <h6 className="border-bottom pb-2 mb-0">App Store</h6>
        {data.map(app => {
          return (
            <StoreEntry
              events={events}
              app={app}
              key={app.digest}
            />
          )
        })}
      </div>
    )
  }

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

function StoreEntry({events = [], app}) {
  const navigate = useNavigate();
  const [installApp] = useInstallAppMutation();
  const [deleteApp] = useDeleteAppMutation();
  const [status, setStatus] = useState(app.installed ? AppInstallStatus.INSTALLED : AppInstallStatus.DEFAULT);

  if (events.length > 0) {
    const latestEvent = events.at(-1)["appInstalled"];
    if (latestEvent && latestEvent.name === app.name) {
        if (status === AppInstallStatus.INSTALLING) {
          setStatus(AppInstallStatus.INSTALLED);
        }
    }
  }

  const handleAppInstallClick = (app) => {
    setStatus(AppInstallStatus.INSTALLING);
    installApp({app});
  }

  const handleAppUninstallClick = (app) => {
    setStatus(AppInstallStatus.DEFAULT);
    deleteApp(app.name);
  }

  const handleAppOpenClick = () => {
    navigate('/home');
  }

  const rowStyles = {
    paddingLeft: "2rem",
  }

  const btnStyles = {
    marginTop: "-2.5rem",
  }

  const descriptionStyles = {
    marginTop: ".50rem",
  }

  return (
    <div className="d-flex text-body-secondary pt-3">

        <img src={app.icon} width={48} height={48}/>

        <div className="pb-3 mb-0 small lh-sm border-bottom w-100 position-relative" style={rowStyles}>
          <div stype={descriptionStyles} >
            Version: { app.version }
          </div>
          <div style={descriptionStyles} dangerouslySetInnerHTML={{__html: marked.parse(app.readme)}} />

          <div>
            {status === AppInstallStatus.DEFAULT && (
              <button
                className="btn btn-outline-primary float-end btn-sm"
                style={btnStyles}
                onClick={() => handleAppInstallClick(app)}>
                  Install
              </button>
            )}
            {status === AppInstallStatus.INSTALLED && (
              <button
                className="btn btn-warning float-end btn-sm"
                onClick={() => handleAppUninstallClick(app)}>
                  Uninstall
              </button>
            )}
          </div>

          <div>
            {status === AppInstallStatus.INSTALLING && (
              <ProgressBar animated now={100} />
            )}
          </div>

        </div>
    </div>
  )
}

function SearchAppEntries() {
  return (
    <div className="my-3 p-3 bg-body rounded shadow-sm">
      <form className="d-flex">
        <input
          className="form-control me-2"
          type="search"
          placeholder="Search"
          aria-label="Search" />
        <button
          className="btn btn-outline-success"
          type="submit">Search</button>
      </form>
    </div>
  )
}