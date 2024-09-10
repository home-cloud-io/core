import * as React from "react";
import { useState, useEffect } from "react";
import { useDispatch } from "react-redux";
import { useNavigate, redirect } from 'react-router-dom';

import Toast from 'react-bootstrap/Toast';
import ToastContainer from 'react-bootstrap/ToastContainer';

import { setUser } from "../../services/web_slice";
import { useInitDeviceMutation, useGetIsDeviceSetupQuery } from "../../services/web_rpc";

import "./DeviceOnboard.css";

function Welcome({ navigate }) {
  return (
    <div className="tab-pane fade show active">
      <h3>Welcome to Home Cloud!</h3>
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
  const [isFormDirty, setFormDirty] = useState(false);
  const [isUsernameValid, setUsernameValidity] = useState(false);
  const [isPasswordValid, setPasswordValidity] = useState(false);

  useEffect(() => {
    if (username.length >= 4) {
      setUsernameValidity(true);
    }

    if (password.length >= 4) {
      setPasswordValidity(true);
    }
  }, [username, password]);

  const handleSubmit = (e) => {
    e.preventDefault();

    if (username.length < 4) {
      setUsernameValidity(false);
    }

    if (password.length < 4) {
      setPasswordValidity(false);
    }

    if (isPasswordValid && isUsernameValid) {
      dispatch(setUser({ username, password }));
      navigate(2);
    } else {
      return;
    }
  }

  const handleUsernameChange = (e) => {
    e.preventDefault();
    const username = e.target.value;
    setFormDirty(true);
    setUsername(username);

    if (username.length < 4) {
      setUsernameValidity(false);
    } else {
      setUsernameValidity(true);
    } 
  }

  const handlePasswordChange = (e) => {
    e.preventDefault();
    const password = e.target.value;
    setFormDirty(true);
    setPassword(password);

    if (password.length < 4) {
      setPasswordValidity(false);
    } else {
      setPasswordValidity(true);
    }
  }

  return (
    <div className="tab-pane fade show active">
      <p>Setup the default administrative user. Don't worry you can always change it later.</p>
      <form className="row g-3 needs-validation">
          <div className="col-12">
            <label className="form-label" htmlFor="usernameValidation">Username</label>
            <input
              id="usernameValidation"
              className={`form-control ${isFormDirty ? (isUsernameValid ? "is-valid" : "is-invalid") : ""}`}
              type="text"
              placeholder="Username"
              value={username}
              onChange={e => handleUsernameChange(e)}
              required />

            <div className={`invalid-feedback ${isFormDirty ? (isUsernameValid ? "d-none" : "") : ""}`}>Username must be at least 4 characters long</div>
          </div>

          <div className="col-12">
            <input
              className={`form-control ${isFormDirty ? (isPasswordValid ? "is-valid" : "is-invalid") : ""}`}
              type="password"
              placeholder="Password"
              value={password}
              onChange={e => handlePasswordChange(e)} />

            <div className={`invalid-feedback ${isFormDirty ? (isPasswordValid ? "d-none" : ""): ""}`}>Password must be at least 4 characters long</div>
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
  const [isFormDirty, setFormDirty] = useState(false);
  const [isTimezoneValid, setTimezoneValidity] = useState(false);

  const handleSubmit = (e) => {
    e.preventDefault();
    useInitDevice();
  }

  const handleSetTimezone = (e) => {
    e.preventDefault();
    const timezone = e.target.value;
    setFormDirty(true);
    setTimezone(timezone);

    if (timezone === "NONE") {
      setTimezoneValidity(false);
    } else {
      setTimezoneValidity(true);
    }
  }

  return (
    <div className="tab-pane fade show active">
      <p>Configure the server</p>

      <form className="row g-3">
        <div className="col-12"> 
          <select
            className={`form-select ${isFormDirty ? (isTimezoneValid ? "is-valid" : "is-invalid"): ""}`}
            value={timezone}
            onChange={e => handleSetTimezone(e)}>
              <option disabled value="NONE">Select a timezone...</option>
              <option value="America/New_York">Eastern (US)</option>
              <option value="America/Chicago">Central (US)</option>
              <option value="America/Denver">Mountain (US)</option>
              <option value="America/Los_Angeles">Pacific (US)</option>
          </select>

          <div className={`invalid-feedback ${isFormDirty ? (isTimezoneValid? "d-none" : ""): ""}`}>Please select a valid timezone</div>
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
  const { data, error, isLoading, refetch } = useGetIsDeviceSetupQuery();

  const [pageNum, setValue] = useState(0);
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const [timezone, setTimezone] = useState("NONE");
  const [autoUpdateApps, setAutoUpdateApps] = useState(true);
  const [autoUpdateOs, setAutoUpdateOs] = useState(true);

  useEffect(() => {
    if (!isLoading && data) {
      if (data.isDeviceSetup) {
        navigate('/store');
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
      })
      refetch()
  }

  return (
    <div>

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

      <ToastContainer className="p-3" position="bottom-center" style={{zIndex: 1}}>
        <Toast show={result.isError} onClose={() => result.reset()}>
          <Toast.Header>
            <strong className="me-auto">Server Error</strong>
            <small></small>
          </Toast.Header>
          <Toast.Body>{result.error}</Toast.Body>
        </Toast>
      </ToastContainer>

    </div>
  );
}