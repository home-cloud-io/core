import * as React from 'react';
import { useDispatch, shallowEqual, useSelector } from 'react-redux';
import { useNavigate } from 'react-router-dom';
import ProgressBar from 'react-bootstrap/ProgressBar';

import {
  useInstallAppMutation,
  useGetAppStoreEntitiesQuery
} from '../../services/web_rpc';
import { marked } from 'marked';
import { AppInstallStatus, setAppInstallStatus } from '../../services/web_slice';

export default function AppStorePage() {
  const { data, error, isLoading } = useGetAppStoreEntitiesQuery();
  const installStatus = useSelector(state => state.server.app_install_status, shallowEqual);

  const ListEntries = () => {
    return (
      <div className="my-3 p-3 bg-body rounded shadow-sm">
        <h6 className="border-bottom pb-2 mb-0">Applications</h6>
        {data.map(app => {
          return (
            <StoreEntry
              app={app}
              key={app.digest}
              status={installStatus[app.name]} />
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

function StoreEntry({app, status = AppInstallStatus.DEFAULT}) {
  const dispatch = useDispatch();
  const navigate = useNavigate();
  const [installApp, result] = useInstallAppMutation();

  const handleAppInstallClick = (app) => {
    status = AppInstallStatus.INSTALLING;
    dispatch(setAppInstallStatus({app, status}));
    installApp({app});
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
          </div>

          <div>
            {status === AppInstallStatus.INSTALLING && (
              <ProgressBar animated now={100} />
            )}
          </div>

          <div>
            {status === AppInstallStatus.INSTALLED && (
              <button
                className="btn btn-outline-success float-end btn-sm"
                style={btnStyles}
                onClick={() => handleAppOpenClick()}
                >
                 Open
              </button>
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