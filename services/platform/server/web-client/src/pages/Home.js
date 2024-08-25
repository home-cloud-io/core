import * as React from 'react';
import { useState } from 'react';
import {
  deleteApp,
  installApp,
  restart,
  shutdown,
  updateApp,
} from '../services/web_rpc';

import { useSelector } from "react-redux";
import { useNavigate } from 'react-router-dom';

import "./Home.css";

export default function Home() {
  const [isNavCollapsed, setIsNavCollapsed] = useState(true);
  const [isSecondaryNavVisible, setSecondaryNavVisibility] = useState(false);
  const [isAlertVisible, setAlertVisibility] = useState(false);

  const userSettings = useSelector((state) => state.user_settings);
  const navigate = useNavigate();
  
  // TODO: Address warning about calling during `useEffect`
  if (userSettings.token === "") {
    navigate('/login');
  }

  const onClickNavCollapseBtn = () => {
    setIsNavCollapsed(!isNavCollapsed);
  }

  return (
    <>
      <svg xmlns="http://www.w3.org/2000/svg" className="d-none">
        <symbol id="check2" viewBox="0 0 16 16">
          <path d="M13.854 3.646a.5.5 0 0 1 0 .708l-7 7a.5.5 0 0 1-.708 0l-3.5-3.5a.5.5 0 1 1 .708-.708L6.5 10.293l6.646-6.647a.5.5 0 0 1 .708 0z"/>
        </symbol>
        <symbol id="circle-half" viewBox="0 0 16 16">
          <path d="M8 15A7 7 0 1 0 8 1v14zm0 1A8 8 0 1 1 8 0a8 8 0 0 1 0 16z"/>
        </symbol>
        <symbol id="moon-stars-fill" viewBox="0 0 16 16">
          <path d="M6 .278a.768.768 0 0 1 .08.858 7.208 7.208 0 0 0-.878 3.46c0 4.021 3.278 7.277 7.318 7.277.527 0 1.04-.055 1.533-.16a.787.787 0 0 1 .81.316.733.733 0 0 1-.031.893A8.349 8.349 0 0 1 8.344 16C3.734 16 0 12.286 0 7.71 0 4.266 2.114 1.312 5.124.06A.752.752 0 0 1 6 .278z"/>
          <path d="M10.794 3.148a.217.217 0 0 1 .412 0l.387 1.162c.173.518.579.924 1.097 1.097l1.162.387a.217.217 0 0 1 0 .412l-1.162.387a1.734 1.734 0 0 0-1.097 1.097l-.387 1.162a.217.217 0 0 1-.412 0l-.387-1.162A1.734 1.734 0 0 0 9.31 6.593l-1.162-.387a.217.217 0 0 1 0-.412l1.162-.387a1.734 1.734 0 0 0 1.097-1.097l.387-1.162zM13.863.099a.145.145 0 0 1 .274 0l.258.774c.115.346.386.617.732.732l.774.258a.145.145 0 0 1 0 .274l-.774.258a1.156 1.156 0 0 0-.732.732l-.258.774a.145.145 0 0 1-.274 0l-.258-.774a1.156 1.156 0 0 0-.732-.732l-.774-.258a.145.145 0 0 1 0-.274l.774-.258c.346-.115.617-.386.732-.732L13.863.1z"/>
        </symbol>
        <symbol id="sun-fill" viewBox="0 0 16 16">
          <path d="M8 12a4 4 0 1 0 0-8 4 4 0 0 0 0 8zM8 0a.5.5 0 0 1 .5.5v2a.5.5 0 0 1-1 0v-2A.5.5 0 0 1 8 0zm0 13a.5.5 0 0 1 .5.5v2a.5.5 0 0 1-1 0v-2A.5.5 0 0 1 8 13zm8-5a.5.5 0 0 1-.5.5h-2a.5.5 0 0 1 0-1h2a.5.5 0 0 1 .5.5zM3 8a.5.5 0 0 1-.5.5h-2a.5.5 0 0 1 0-1h2A.5.5 0 0 1 3 8zm10.657-5.657a.5.5 0 0 1 0 .707l-1.414 1.415a.5.5 0 1 1-.707-.708l1.414-1.414a.5.5 0 0 1 .707 0zm-9.193 9.193a.5.5 0 0 1 0 .707L3.05 13.657a.5.5 0 0 1-.707-.707l1.414-1.414a.5.5 0 0 1 .707 0zm9.193 2.121a.5.5 0 0 1-.707 0l-1.414-1.414a.5.5 0 0 1 .707-.707l1.414 1.414a.5.5 0 0 1 0 .707zM4.464 4.465a.5.5 0 0 1-.707 0L2.343 3.05a.5.5 0 1 1 .707-.707l1.414 1.414a.5.5 0 0 1 0 .708z"/>
        </symbol>
      </svg>

      <div className="dropdown position-fixed bottom-0 end-0 mb-3 me-3 bd-mode-toggle">
        <button className="btn btn-bd-primary py-2 dropdown-toggle d-flex align-items-center"
                id="bd-theme"
                type="button"
                aria-expanded="false"
                data-bs-toggle="dropdown"
                aria-label="Toggle theme (auto)">
          <svg className="bi my-1 theme-icon-active" width="1em" height="1em"><use href="#circle-half"></use></svg>
          <span className="visually-hidden" id="bd-theme-text">Toggle theme</span>
        </button>
        <ul className="dropdown-menu dropdown-menu-end shadow" aria-labelledby="bd-theme-text">
          <li>
            <button type="button" className="dropdown-item d-flex align-items-center" data-bs-theme-value="light" aria-pressed="false">
              <svg className="bi me-2 opacity-50" width="1em" height="1em"><use href="#sun-fill"></use></svg>
              Light
              <svg className="bi ms-auto d-none" width="1em" height="1em"><use href="#check2"></use></svg>
            </button>
          </li>
          <li>
            <button type="button" className="dropdown-item d-flex align-items-center" data-bs-theme-value="dark" aria-pressed="false">
              <svg className="bi me-2 opacity-50" width="1em" height="1em"><use href="#moon-stars-fill"></use></svg>
              Dark
              <svg className="bi ms-auto d-none" width="1em" height="1em"><use href="#check2"></use></svg>
            </button>
          </li>
          <li>
            <button type="button" className="dropdown-item d-flex align-items-center active" data-bs-theme-value="auto" aria-pressed="true">
              <svg className="bi me-2 opacity-50" width="1em" height="1em"><use href="#circle-half"></use></svg>
              Auto
              <svg className="bi ms-auto d-none" width="1em" height="1em"><use href="#check2"></use></svg>
            </button>
          </li>
        </ul>
      </div>

      <nav className="navbar navbar-expand-lg fixed-top navbar-dark bg-dark"
        aria-label="Main navigation">
          <div className="container-fluid">
            <div>
              <a className="navbar-brand" href="#">Home Cloud</a>
            </div>

            <div>
            <button 
                id="navbarSideCollapse"
                className="navbar-toggler p-0 border-0"
                type="button"
                aria-label="Toggle navigation"
                onClick={onClickNavCollapseBtn}>
                  <span className="navbar-toggler-icon"></span>
              </button>
            <div className={`navbar-collapse offcanvas-collapse ${isNavCollapsed ? '' : 'open'}`} id="navbarsExampleDefault">
              <ul className="navbar-nav me-auto mb-2 mb-lg-0">
                {/* <li className="nav-item">
                  <a className="nav-link active" aria-current="page" href="#">Home</a>
                </li> */}
                {/* <li className="nav-item">
                  <a className="nav-link" href="#">Device Details</a>
                </li>
                <li className="nav-item">
                  <a className="nav-link" href="#">Profile</a>
                </li>
                <li className="nav-item">
                  <a className="nav-link" href="#">Switch account</a>
                </li> */}
                <li className="nav-item dropdown">
                  <a className="nav-link dropdown-toggle" href="#" data-bs-toggle="dropdown" aria-expanded="false">Settings</a>
                  <ul className="dropdown-menu">
                    <li><a className="dropdown-item" href="#">Settings</a></li>
                    <li><a className="dropdown-item" href="#">Logout</a></li>
                    <li><a className="dropdown-item" href="#">Shutdown</a></li>
                    <li><a className="dropdown-item" href="#">Restart</a></li>
                  </ul>
                </li>
              </ul>
            </div>

            </div>
        </div>
      </nav>

    {isSecondaryNavVisible && <SecondaryNav />}

      <main className="container">
        {isAlertVisible && <Alert />}

        <DeviceDetails />
        <InstalledApplicationsList />

      </main>

      {/* <div>
        <button onClick={() => shutdown()}>Shutdown Host</button>{' '}
        <button onClick={() => restart()}>Restart Host</button>
        <br></br>
        Hello World
        <br></br>
        <button onClick={() => installApp('hello-world')}>Install App</button>
        {'  '}
        <button onClick={() => deleteApp('hello-world')}>Delete App</button>
        {'  '}
        <button onClick={() => updateApp('hello-world')}>Update App</button>
        <br></br>
        Postgres
        <br></br>
        <button onClick={() => installApp('postgres')}>Install App</button>
        {'  '}
        <button onClick={() => deleteApp('postgres')}>Delete App</button>
        {'  '}
        <button onClick={() => updateApp('postgres')}>Update App</button>
        <br></br>
        Immich
        <br></br>
        <button onClick={() => installApp('immich')}>Install App</button>
        {'  '}
        <button onClick={() => deleteApp('immich')}>Delete App</button>
        {'  '}
        <button onClick={() => updateApp('immich')}>Update App</button>
        <br></br>
      </div> */}
    </>
  );
}

function InstalledApplicationsList() {
  return (
    <div>

      <div className="my-3 p-3 bg-body rounded shadow-sm">
        <h6 className="border-bottom pb-2 mb-0">Applications</h6>

        <div className="">
          <Application />
          <Application />
          <Application />
        </div>

        <small className="d-block text-end mt-3">
          <a href="#">All Applications</a>
        </small>
      </div>
    </div>
  )
}

function Application() {
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
              <strong className="d-block text-gray-dark">Online</strong>
            </p>
          </div>

        </div>
    </div>
  )
}

function DeviceDetails() {
  const styles = {
    float: "right",
    marginTop: "-2.75rem",
  }

  return (
    <div className="my-3 p-3 bg-body rounded shadow-sm">
      <div className="border-bottom">
      <h6 className="pb-2 mb-0">Device Details</h6>

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

function SecondaryNav() {
  return (
    <div className="nav-scroller bg-body shadow-sm" id="secondary-menu">
      <nav className="nav" aria-label="Secondary navigation">
        <a className="nav-link active" aria-current="page" href="#">Dashboard</a>
        <a className="nav-link" href="#">
          Friends
          <span className="badge text-bg-light rounded-pill align-text-bottom">27</span>
        </a>
        <a className="nav-link" href="#">Explore</a>
        <a className="nav-link" href="#">Suggestions</a>
        <a className="nav-link" href="#">Link</a>
        <a className="nav-link" href="#">Link</a>
        <a className="nav-link" href="#">Link</a>
        <a className="nav-link" href="#">Link</a>
        <a className="nav-link" href="#">Link</a>
      </nav>
    </div>
  )
}

function Alert() {
  return (
    <div className="d-flex align-items-center p-3 my-3 text-white bg-purple rounded shadow-sm">
      <img className="me-3" src="../assets/brand/bootstrap-logo-white.svg" alt="" width="48" height="38" />
      <div className="lh-1">
        <h1 className="h6 mb-0 text-white lh-1">Bootstrap</h1>
        <small>Since 2011</small>
      </div>
    </div>
  )
}