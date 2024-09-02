import * as React from 'react';
import { 
  useGetAppsHealthCheckQuery, 
  useGetSystemStatsQuery,
  useDeleteAppMutation,
} from '../../services/web_rpc';

export default function HomePage() {
  return (
    <div> 
        <InstalledApplicationsList />
        <DeviceDetails />
    </div>
  )
}

export function InstalledApplicationsList() {
  const { data, error, isLoading } = useGetAppsHealthCheckQuery();

  const ListEntries = () => {
    if (data.checks.length === 0) {
      return <p>No applications installed</p>
    }

    return (
      <div>
        {data.checks.map(app => {
          return (
            <Application app={app} key={app.name}/>
          )
        })}
      </div>
    )
  }

  return (
    <div>

      <div className="my-3 p-3 bg-body rounded shadow-sm">
        <h6 className="border-bottom pb-2 mb-0">Installed Applications</h6>

          {isLoading ? (
            <p>Loading...</p>
          ) : error ? (
            <p>Error: {error}</p>
          ) : (
            <ListEntries />
          )}


        {/* <small className="d-block text-end mt-3">
          <a href="#">All Applications</a>
        </small> */}
      </div>
    </div>
  )
}

function Application({app}) {
  const [deleteApp, result] = useDeleteAppMutation();
  const descriptionStyles = {
    marginTop: ".50rem",
  }

  const uninstallBtnStyles = {
    marginTop: "-3rem"
  }

  return (
    <div className="d-flex text-body-secondary pt-3">
        <img src={app.display.iconUrl} width={48} height={48}/>

        <div className="pb-3 mb-0 small lh-sm border-bottom w-100 position-relative">
          <div className="d-flex justify-content-between">
            <strong className="text-gray-dark">{app.name}</strong>
          </div>

          <div style={descriptionStyles}>
            <p>{app.display.description}</p>
          </div>

          <div className="float-end" style={uninstallBtnStyles}>
            <button className="btn btn-warning" onClick={() => deleteApp(app.name)}>Uninstall</button> 
          </div>
        </div>
    </div>
  )
}

export function DeviceDetails() {
  const { data, error, isLoading } = useGetSystemStatsQuery();

  const styles = {
    float: "right",
    marginTop: "-2.75rem",
  }

  const formatBytes = (bytes, decimals = 2) => {
    if (bytes === 0) return '0 Bytes';

    const k = 1024;
    const dm = decimals < 0 ? 0 : decimals;
    const sizes = ['Bytes', 'KB', 'MB', 'GB', 'TB', 'PB', 'EB', 'ZB', 'YB'];

    const i = Math.floor(Math.log(bytes) / Math.log(k));

    return parseFloat((bytes / Math.pow(k, i)).toFixed(dm)) + ' ' + sizes[i];
  }

  const formatPercentage = (free, total) => { 
    return Math.round(((total-free)/ total) * 100);
  }

  return (
    <div className="my-3 p-3 bg-body rounded shadow-sm">
      <div className="border-bottom">
      <h6 className="pb-2 mb-0">Server Status</h6>

      <div className="d-flex text-body-secondary pt-3" style={styles}>
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
          <strong className="d-block text-gray-dark">Online</strong>
        </p>
        <br />
      </div>

      </div>

      <div className="d-flex text-body-secondary pt-3">
        <p className="mb-0 small lh-sm border-bottom">
          <strong className="d-block text-gray-dark">Storage</strong>
          {isLoading ? (
            <p>Loading...</p>
          ) : error ? (
            <p>Error: {error}</p>
          ) : (
            <p>{formatBytes(data.drives[0].freeBytes)} free of {formatBytes(data.drives[0].totalBytes)}</p>
          )}
        </p>
      </div>

      <div className="progress-stacked">
          {isLoading ? (
            <p>Loading...</p>
          ) : error ? (
            <p>Error: {error}</p>
          ) : (
            <div
              className="progress"
              role="progressbar"
              aria-label="Segment one"
              aria-valuenow="15"
              aria-valuemin="0"
              aria-valuemax="100"
              style={{width: formatPercentage(data.drives[0].freeBytes, data.drives[0].totalBytes)}}>
              <div className="progress-bar bg-warning"></div>
            </div>
          )}
      </div>

      <div>
        <div className="">
        <svg
            className="bd-placeholder-img flex-shrink-0 me-2 rounded"
            width="8"
            height="8"
            xmlns="http://www.w3.org/2000/svg"
            role="img"
            aria-label="Placeholder: 32x32"
            preserveAspectRatio="xMidYMid slice"
            focusable="false">
              <title>Placeholder</title>
              <rect width="100%" height="100%" fill="#ffc107"/><text x="50%" y="50%" fill="#ffc107" dy=".3em">32x32</text>
          </svg>
        <label>Drive Space Used</label>
        </div>
      </div>

      {/* <small className="d-block text-end mt-3">
        <a href="#">Device Details</a>
      </small> */}
    </div>
  )
}