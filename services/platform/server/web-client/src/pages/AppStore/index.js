import * as React from 'react';

import { 
  useInstallAppMutation,
  useGetAppStoreEntitiesQuery
} from '../../services/web_rpc';
import { marked } from 'marked';

export default function AppStorePage() {
  const { data, error, isLoading } = useGetAppStoreEntitiesQuery(); 

  const ListEntries = () => {
    return (
      <div className="my-3 p-3 bg-body rounded shadow-sm">
        <h6 className="border-bottom pb-2 mb-0">Applications</h6>
        {data.map(app => {
          return (
            <StoreEntry app={app} key={app.digest}/>
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

function StoreEntry({app}) {
  const [installApp, result] = useInstallAppMutation();

  const rowStyles = {
    paddingLeft: "2rem",
  }

  const btnStyles = {
    marginTop: "-2.5rem",
  }

  const descriptionStyles = {
    marginTop: ".50rem",
  }

  const onAppClick = (app) => {
    console.log(`App clicked: ${app}`);
  }

  const onMouseOver = (app) => {
    console.log(`App entered: ${app}`);
  }

  return (
    <div className="d-flex text-body-secondary pt-3">

        <img src={app.icon} width={48} height={48}/>

        <div className="pb-3 mb-0 small lh-sm border-bottom w-100 position-relative" style={rowStyles}>
          <div stype={descriptionStyles} >
            Version: { app.version }
          </div>
          <div style={descriptionStyles} dangerouslySetInnerHTML={{__html: marked.parse(app.readme)}} />
          <button 
            className="btn btn-outline-primary float-end btn-sm"
            style={btnStyles}
            onClick={() => installApp(app)}>
              Install
          </button>

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