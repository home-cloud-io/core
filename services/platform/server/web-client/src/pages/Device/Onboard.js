import * as React from "react";
import { useState } from "react";
import { useDispatch, useSelector } from "react-redux";
import { useNavigate } from 'react-router-dom';
import { setUser } from "../../services/web_slice";
import { useInitDeviceMutation } from "../../services/web_rpc";

import "./DeviceOnboard.css";

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
              checked={autoUpdateApps}
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
  const dispatch = useDispatch();
  const navigate = useNavigate();
  const [initDevice, result] = useInitDeviceMutation();

  const [pageNum, setValue] = React.useState(0);
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const [timezone, setTimezone] = useState("");
  const [autoUpdateApps, setAutoUpdateApps] = useState(true);
  const [autoUpdateOs, setAutoUpdateOs] = useState(true);

  const handleClick = (val) => setValue(val);

  const initServer = () => {
    dispatch(initDevice({
      username: username,
      password: password,
      timezone: timezone,
      autoUpdateApps: autoUpdateApps,
      autoUpdateOs: autoUpdateOs,
    }));

  }

  if (result.error) {
    console.log("Error initializing device");
  }

  if (result.data) {
    navigate('/store');
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