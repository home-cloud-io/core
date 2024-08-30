import * as React from 'react';
import { useGetAppsHealthCheckQuery } from '../../services/web_rpc';

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
            <p>Error: {error.message}</p>
          ) : (
            <ListEntries />
          )}


        <small className="d-block text-end mt-3">
          <a href="#">All Applications</a>
        </small>
      </div>
    </div>
  )
}

function Application({app}) {
  return (
    <div className="d-flex text-body-secondary pt-3">
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
            <strong className="text-gray-dark">{app.name}</strong>
          </div>
        </div>
    </div>
  )
}

export function DeviceDetails() {
  const styles = {
    float: "right",
    marginTop: "-2.75rem",
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
        <p className="pb-3 mb-0 small lh-sm border-bottom">
          <strong className="d-block text-gray-dark">Storage</strong>
          655.5 GB of 1 TB used
        </p>
      </div>

      <div className="progress-stacked">
        <div className="progress" role="progressbar" aria-label="Segment one" aria-valuenow="15" aria-valuemin="0" aria-valuemax="100" style={{width: "15%"}}>
          <div className="progress-bar bg-warning"></div>
        </div>
        <div className="progress" role="progressbar" aria-label="Segment two" aria-valuenow="30" aria-valuemin="0" aria-valuemax="100" style={{width: "30%"}}>
          <div className="progress-bar bg-danger"></div>
        </div>
        <div className="progress" role="progressbar" aria-label="Segment three" aria-valuenow="20" aria-valuemin="0" aria-valuemax="100" style={{width: "20%"}}>
          <div className="progress-bar bg-info"></div>
        </div>
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
        <label>OS</label>
        </div>

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
                <rect width="100%" height="100%" fill="#dc3545"/><text x="50%" y="50%" fill="#dc3545" dy=".3em">32x32</text>
            </svg>
          <label>Applications</label>
        </div>

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
                <rect width="100%" height="100%" fill="#0dcaf0"/><text x="50%" y="50%" fill="#0dcaf0" dy=".3em">32x32</text>
            </svg>
          <label>Files</label>
        </div>

      </div>

      <small className="d-block text-end mt-3">
        <a href="#">Device Details</a>
      </small>
    </div>
  )
}