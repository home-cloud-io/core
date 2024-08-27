import * as React from "react";
import { useState } from "react";
import { useDispatch, useSelector } from "react-redux";

import "./DeviceOnboard.css";

import {
  setUser,
  setDeviceSettings,
  setDefaultApps
} from "../../services/web_slice";

import { useInitDeviceMutation } from "../../services/web_rpc";

function Welcome({ navigate }) {
  return (
    <div className="tab-pane fade show active">
      <h3>Welcome To Home Cloud</h3>
      <p>Home Cloud is a personal cloud solution that allows you to store and access your data from anywhere in the world. It is a secure and private cloud solution that is easy to use and provides you with all the features you need to manage your data.</p>
 
      <div className="col-12">
        <button 
          style={{float:"right"}}
          className="btn btn-outline-primary"
          type="button"
          onClick={() => navigate(1)}>Next</button>
      </div>
    </div>
  );
}

function UserSetup({ navigate, setUser }) {
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const dispatch = useDispatch();

  const handleSubmit = (e) => {
    e.preventDefault();
    dispatch(setUser({ username, password }));
    navigate(2);
  }

  return (
    <div className="tab-pane fade show active">
      <p>Setup the default administrative user. Don't worry you can always change it later.</p>
      <form className="row g-3">
          <div className="col-12">
            <input
              className="form-control"
              type="text"
              placeholder="Username"
              value={username}
              onChange={e => setUsername(e.target.value)} />
          </div>
          
          <div className="col-12">
            <input
              className="form-control"
              type="password"
              placeholder="Password"
              value={password}
              onChange={e => setPassword(e.target.value)} />
          </div>

          <div className="col-12">
            <button 
              style={{float:"left"}}
              className="btn btn-outline-primary"
              type="button"
              onClick={() => navigate(0)}>Back</button>
            
            <button 
              style={{float:"right"}}
              className="btn btn-outline-primary"
              type="button"
              onClick={(evt) => handleSubmit(evt)}>Next</button>
          </div>

      </form>
    </div>
  );
}

function DeviceSettings({ navigate, setDeviceSettings }) {
  const dispatch = useDispatch();

  const [timezone, setTimezone] = useState("");
  const [autoUpdateApps, setAutoUpdateApps] = useState(false);
  const [autoUpdateOs, setAutoUpdateOs] = useState(false);

  const handleSubmit = (e) => {
    e.preventDefault();
    dispatch(setDeviceSettings({ timezone, autoUpdateApps, autoUpdateOs }));
    navigate(3);
  }

  return (
    <div className="tab-pane fade show active">
      <p>Configure the server</p>

      <form className="row g-3">
        <div className="col-12"> 
          <select
            className="form-select"
            value={timezone}
            onChange={e => setTimezone(e.target.value)}>
              <option>Select a timezone...</option>
              <option value="1">GMT</option>
              <option value="2">CST</option>
              <option value="3">PST</option>
          </select>
        </div> 

        <div className="col-12">
          <div className="form-check form-switch form-check-reverse">
            <input 
              className="form-check-input"
              type="checkbox"
              role="switch"
              value="true"
              onChange={e => setAutoUpdateApps(e.target.value)}/>
            <label className="form-check-label">Automatically update applications</label>
          </div>
        </div>

        <div className="col-12">
          <div className="form-check form-switch form-check-reverse">
            <input
              className="form-check-input"
              type="checkbox"
              role="switch"
              value="true"
              onChange={e => setAutoUpdateOs(e.target.value)} />
            <label className="form-check-label">Automatically update server</label>
          </div>
        </div>

        <div className="col-12">
          <button 
            style={{float:"left"}}
            className="btn btn-outline-primary"
            type="button"
            onClick={() => navigate(1)}>Back</button>
          
          <button 
            style={{float:"right"}}
            className="btn btn-outline-primary"
            type="button"
            onClick={e => handleSubmit(e)}>Next</button>
        </div>

      </form>
    </div>
  );
} 

function DefaultApplications({ navigate, setDefaultApps, useInitDevice }) {
  const deviceSetup = useSelector((state) => state.server);
  const dispatch = useDispatch();
  const [defaultApps, setSelectedApps] = useState([]);

  const handleSubmit = (e) => {
    e.preventDefault();
    dispatch(setDefaultApps(defaultApps));
    dispatch(useInitDevice());
  }

  const handleClick = (e) => {
    const { value } = e.target;
    if (defaultApps.includes(value)) {
      setSelectedApps(defaultApps.filter(app => app !== value));
    } else {
      setSelectedApps([...defaultApps, value]);
    }
  }

  return (
    <div className="tab-pane fade show active">
      <p>Choose some applications to install while the device is setup.</p>

      <form className="row g-3">
        <ul className="list-group">

          <li className="list-group-item d-flex justify-content-between align-items-start">
            <div className="ms-2 me-auto">
              <div className="fw-bold">Immich</div>
              Your personal image gallery
            </div>
            <input
              className="form-check-input me-1"
              type="checkbox"
              value="immich"
              id="firstCheckbox"
              onClick={e => handleClick(e)}></input>
          </li>

          <li className="list-group-item d-flex justify-content-between align-items-start">
            <div className="ms-2 me-auto">
              <div className="fw-bold">Immich</div>
              Your personal image gallery
            </div>
            <input
              className="form-check-input me-1"
              type="checkbox"
              value="app2"
              id="firstCheckbox"
              onClick={e => handleClick(e)}></input>
          </li>

          <li className="list-group-item d-flex justify-content-between align-items-start">
            <div className="ms-2 me-auto">
              <div className="fw-bold">Immich</div>
              Your personal image gallery
            </div>
            <input
              className="form-check-input me-1"
              type="checkbox"
              value="app3"
              id="firstCheckbox"
              onClick={e => handleClick(e)}></input>
          </li>
        </ul>
      
        <div className="col-12">
          <button 
            style={{float:"left"}}
            className="btn btn-outline-primary"
            type="button"
            onClick={() => navigate(2)}>Back</button>

          <button 
            style={{float:"right"}}
            className="btn btn-outline-primary"
            type="button"
            onClick={e => handleSubmit(e)}>Setup</button>
        </div>

      </form>
    </div>
  );
}

export default function DeviceOnboardPage() {
  const [value, setValue] = React.useState(0);
  const [initDevice, result] = useInitDeviceMutation();

  const handleClick = (val) => setValue(val);

  return (
    <>
      <div className="container card shadow d-flex justify-content-center">
        <ul className="nav nav-pills mb-12 shadow-sm" id="pills-tab" role="tablist">
          <li className="nav-item">
            <a 
              className={`nav-link ${value === 0 ? 'active' : ''}`}
              id="pills-home-tab"
              onClick={() => handleClick(0)}>Getting Started</a>
          </li>
          <li className="nav-item">
            <a 
              className={`nav-link ${value === 1 ? 'active' : ''}`}
              id="pills-home-tab"
              onClick={() => handleClick(1)}>User Setup</a>
          </li>
          <li className="nav-item">
            <a 
              className={`nav-link ${value === 2 ? 'active' : ''}`}
              id="pills-home-tab"
              onClick={() => handleClick(2)}>Device Settings</a>
          </li>
          <li className="nav-item">
            <a 
              className={`nav-link ${value === 3 ? 'active' : ''}`}
              id="pills-home-tab"
              onClick={() => handleClick(3)}>Default Applications</a>
          </li>
        </ul>

        <div className="tab-content" id="pills-tabContent p-3">
          {value == 0 && <Welcome navigate={handleClick}/>}
          {value == 1 && <UserSetup navigate={handleClick} setUser={setUser} />}
          {value == 2 && <DeviceSettings navigate={handleClick} setDeviceSettings={setDeviceSettings}/>}
          {value == 3 && <DefaultApplications navigate={handleClick} setDefaultApps={setDefaultApps} useInitDevice={initDevice}/>}
        </div>
      </div>
    </>
  );
}