import * as React from "react";
import { useState, useEffect } from "react";
import { useDispatch } from "react-redux";
import { useNavigate, redirect } from 'react-router-dom';
import { setUser } from "../../services/web_slice";
import { useInitDeviceMutation, useGetIsDeviceSetupQuery } from "../../services/web_rpc";

import "./DeviceOnboard.css";

function Welcome({ navigate }) {
  return (
    <div className="tab-pane fade show active">
      <h3>Welcome To Home Cloud</h3>
      <p>The easy-to-use solution that enables you to say goodbye to the high-cost, privacy nightmare of Big Tech services so that you can finally take back control over your digital life!</p>

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

function UserSetup({ navigate, setUsername, setPassword, username, password}) {
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

function DeviceSettings({ navigate, useInitDevice, setTimezone, setAutoUpdateApps, setAutoUpdateOs, timezone, autoUpdateApps, autoUpdateOs }) {
  const handleSubmit = (e) => {
    e.preventDefault();
    useInitDevice();
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
              <option value="America/New_York">Eastern (US)</option>
              <option value="America/Chicago">Central (US)</option>
              <option value="America/Denver">Mountain (US)</option>
              <option value="America/Los_Angeles">Pacific (US)</option>
          </select>
        </div> 

        <div className="col-12">
          {/* TODO: enable this when it's configurable later on */}
          <div className="form-check form-switch form-check-reverse" hidden={true}>
            <input 
              className="form-check-input"
              type="checkbox"
              role="switch"
              value="true"
              checked={autoUpdateApps}
              onChange={e => setAutoUpdateApps(e.target.value)}/>
            <label className="form-check-label">Automatically update applications</label>
          </div>
        </div>

        <div className="col-12">
          {/* TODO: enable this when it's configurable later on */}
          <div className="form-check form-switch form-check-reverse" hidden={true}>
            <input
              className="form-check-input"
              type="checkbox"
              role="switch"
              value="true"
              checked={autoUpdateOs}
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

export default function DeviceOnboardPage() {
  const navigate = useNavigate();
  const [initDevice, result] = useInitDeviceMutation();
  const { data, error, isLoading } = useGetIsDeviceSetupQuery();

  const [pageNum, setValue] = useState(0);
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const [timezone, setTimezone] = useState("");
  const [autoUpdateApps, setAutoUpdateApps] = useState(true);
  const [autoUpdateOs, setAutoUpdateOs] = useState(true);

  useEffect(() => {
    if (!isLoading && data) {
      if (data.isDeviceSetup) {
        navigate('/home');
      }
    }
  })

  const handleClick = (val) => setValue(val);

  const initServer = () => {
      const res = initDevice({
        username: username,
        password: password,
        timezone: timezone,
        autoUpdateApps: autoUpdateApps,
        autoUpdateOs: autoUpdateOs,
      }).unwrap();

      res.then((data) => {
        navigate('/store');
      }).catch((error) => {
        console.error(error);
      }); 
  }

  return (
    <>
      <div className="container card shadow d-flex justify-content-center">
        <div className="tab-content" id="pills-tabContent p-3">
          {pageNum == 0 && <Welcome navigate={handleClick}/>}
          {pageNum == 1 && <UserSetup navigate={handleClick} setUsername={setUsername} setPassword={setPassword} username={username} password={password} />}
          {pageNum == 2 && <DeviceSettings
            navigate={handleClick}
            useInitDevice={initServer}
            setTimezone={setTimezone}
            setAutoUpdateApps={setAutoUpdateApps} 
            setAutoUpdateOs={setAutoUpdateOs}
            timezone={timezone}
            autoUpdateApps={autoUpdateApps}
            autoUpdateOs={autoUpdateOs} 
            />}
        </div>
      </div>
    </>
  );
}