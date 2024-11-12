import React, { useState } from 'react';
import { Outlet, NavLink } from 'react-router-dom';
import { useGetEventsQuery } from '../services/web_rpc';
const logo = require('../assets/logo-white-flat.png');

export default function DefaultLayout() {
  const [isNavCollapsed, setIsNavCollapsed] = useState(true);
  // events are not used here (yet) but this is called to initialize the event stream on any page load
  useGetEventsQuery();

  const onClickNavCollapseBtn = () => {
    setIsNavCollapsed(!isNavCollapsed);
  };

  return (
    <>
      <nav
        className="navbar navbar-expand-lg fixed-top navbar-dark"
        aria-label="Main navigation"
      >
        <div className="container-fluid">
          <div>
            <img src={logo} width={53} height={53} />
            <a className="navbar-brand" href="/home">
              Home Cloud
            </a>
          </div>

          <div>
            <button
              id="navbarSideCollapse"
              className="navbar-toggler p-0 border-0"
              type="button"
              aria-label="Toggle navigation"
              onClick={onClickNavCollapseBtn}
            >
              <span className="navbar-toggler-icon"></span>
            </button>
            <div
              className={`navbar-collapse offcanvas-collapse ${
                isNavCollapsed ? '' : 'open'
              }`}
              id="navbarsExampleDefault"
            >
              <ul className="navbar-nav me-auto mb-2 mb-lg-0">
                <li className="nav-item">
                  <NavLink
                    to="/home"
                    className="nav-link"
                    onClick={onClickNavCollapseBtn}
                  >
                    Home
                  </NavLink>
                </li>

                <li className="nav-item">
                  <NavLink
                    to="/store"
                    className="nav-link"
                    onClick={onClickNavCollapseBtn}
                  >
                    Store
                  </NavLink>
                </li>

                <li className="nav-item">
                  <NavLink
                    to="/upload"
                    className="nav-link"
                    onClick={onClickNavCollapseBtn}
                  >
                    Upload
                  </NavLink>
                </li>

                <li className="nav-item">
                  <NavLink
                    to="/settings"
                    className="nav-link"
                    onClick={onClickNavCollapseBtn}
                  >
                    Settings
                  </NavLink>
                </li>
              </ul>
            </div>
          </div>
        </div>
      </nav>

      <main className="container">
        <Outlet />
      </main>
    </>
  );
}
