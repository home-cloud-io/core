import * as React from 'react';

import "./AppStore.css";

import { useGetAppStoreEntitiesQuery } from '../../services/web_rpc';

export default function AppStore() {
  const { apps, error, isLoading } = useGetAppStoreEntitiesQuery(); 

  return (
    <>
      <SearchAppEntries />
        {
          isLoading ? (
            <div>Loading...</div>
          ) : error ? (
            <div>Error: {error.message}</div>
          ) : (
            <div>
              <AppStoreEntries apps={apps}/>
            </div>
          )
        }
    </>
  );
}

export function AppStoreEntries({apps}) {
  return (
    <div>

      <div className="my-3 p-3 bg-body rounded shadow-sm">
        <div className="">
          {apps.map((app) => (
            <StoreEntry details={app} />
          ))}

          {/* <StoreEntry />
          <StoreEntry />
          <StoreEntry /> */}
        </div>

      </div>
    </div>
  )
}

function StoreEntry({details}) {
  const styles = {
    marginTop: ".25rem",
  }

  const onAppClick = (app) => {
    console.log(`App clicked: ${app}`);
  }

  const onMouseOver = (app) => {
    console.log(`App entered: ${app}`);
  }

  return (
    <div className="d-flex text-body-secondary pt-3"
      onClick={() => onAppClick("immich")}
      onMouseOver={() => onMouseOver("immitch")}>
      <svg
        className="bd-placeholder-img flex-shrink-0 me-2 rounded"
        width="64"
        height="64"
        xmlns="http://www.w3.org/2000/svg"
        role="img"
        aria-label="Placeholder: 32x32"
        preserveAspectRatio="xMidYMid slice"
        focusable="false">
          <title>Placeholder</title>
          <rect width="100%" height="100%" fill="#6528e0"/><text x="50%" y="50%" fill="#6528e0" dy=".3em">32x32</text>
        </svg>

        <div className="pb-3 mb-0 small lh-sm border-bottom w-100 position-relative">
          <div className="d-flex justify-content-between">
            <strong className="text-gray-dark">Immich</strong>
          </div>

          <span className="float-end app-version">Version: 1.0.1</span>

          <div className="d-flex text-body-secondary pt-3 float-end position-absolute top-25 end-0" style={styles}>
            <svg
              className="bd-placeholder-img flex-shrink-0 me-2 rounded"
              width="16"
              height="16"
              xmlns="http://www.w3.org/2000/svg"
              role="img"
              aria-label="Placeholder: 32x32"
              preserveAspectRatio="xMidYMid slice"
              focusable="false">
                <title>Placeholder</title>
                <rect width="100%" height="100%" fill="#28e053"/><text x="50%" y="50%" fill="#28e053" dy=".3em">32x32</text>
            </svg>
            <p
              className="pb-3 mb-0 small lh-sm"
              id="deviceStatusIndicatorLabel">
              <strong className="d-block text-gray-dark"></strong>
            </p>
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