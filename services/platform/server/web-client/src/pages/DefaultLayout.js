import React, { useEffect, useState } from 'react';
import { Outlet, NavLink } from "react-router-dom";
import { useDispatch, shallowEqual, useSelector } from 'react-redux';
import { useGetEventsQuery } from '../services/web_rpc';

import "./DefaultLayout.css";

export default function DefaultLayout() {
  const dispatch = useDispatch();
  const [isNavCollapsed, setIsNavCollapsed] = useState(true);
  const [isSecondaryNavVisible, setSecondaryNavVisibility] = useState(false);
  const [isAlertVisible, setAlertVisibility] = useState(false);
  const serverStatus = useSelector(state => state.server.event_stream_connection_status, shallowEqual);
  const { data, error, isLoading } = useGetEventsQuery();

  const onClickNavCollapseBtn = () => {
    setIsNavCollapsed(!isNavCollapsed);
  }

  return (
    <>
      <SVGIcons />
      <OffCanvasNavigation />

      <nav className="navbar navbar-expand-lg fixed-top navbar-dark bg-dark"
        aria-label="Main navigation">
          <div className="container-fluid">
            <div>
              <a className="navbar-brand" href="#">Home Cloud ({serverStatus})</a>
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
                  <li className="nav-item">
                      <NavLink className="nav-link" to="/home" onClick={onClickNavCollapseBtn}>Home</NavLink>
                  </li>

                  <li className="nav-item">
                      <NavLink to="/store" className="nav-link" onClick={onClickNavCollapseBtn}>Store</NavLink>
                  </li>

                  <li className="nav-item">
                      <NavLink to="/upload" className="nav-link" onClick={onClickNavCollapseBtn}>Upload</NavLink>
                  </li>

                  <li className="nav-item">
                      <NavLink to="/settings" className="nav-link" onClick={onClickNavCollapseBtn}>Settings</NavLink>
                  </li>
                </ul>
              </div>
            </div>
        </div>
      </nav>

    {isSecondaryNavVisible && <SecondaryNav />}

      <main className="container">
        {isAlertVisible && <Alert />}
        <Outlet />
      </main>
    </>
  );
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

function OffCanvasNavigation() {
  return (
    <div className="dropdown position-fixed bottom-0 end-0 mb-3 me-3 bd-mode-toggle">
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
  )
}

function SVGIcons() {
  return (
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
  )
}