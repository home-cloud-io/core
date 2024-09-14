import * as React from 'react';

import {
  useShutdownHostMutation,
  useRestartHostMutation,
  useGetDeviceSettingsQuery,
  useSetDeviceSettingsMutation,
} from '../../services/web_rpc';
import { useState, useEffect } from 'react';

import Toast from 'react-bootstrap/Toast';
import ToastContainer from 'react-bootstrap/ToastContainer';

export default function SettingsPage() {
  const [shutdownHost, shutdownResult] = useShutdownHostMutation();
  const [restartHost, restartResult] = useRestartHostMutation();
  const [saveSettings, result] = useSetDeviceSettingsMutation();
  const { data, error, isLoading } = useGetDeviceSettingsQuery();

  // TODO: make this way better
  if (isLoading) {
    return <div>Loading...</div>;
  }

  // TODO: make this way better
  if (error) {
    return <div>Error: {error.message}</div>;
  }

  const headerStyles = {
    paddingTop: '.75rem',
    paddingBottom: '1rem',
  };

  return (
    <div>
      <div className="my-3 p-3 bg-body rounded shadow-sm">
        <div className="float-end">
          <div className="dropdown">
            <button
              className="btn btn-secondary dropdown-toggle"
              type="button"
              data-bs-toggle="dropdown"
              aria-expanded="false"
            >
              <svg
                xmlns="http://www.w3.org/2000/svg"
                width="16"
                height="16"
                fill="currentColor"
                className="bi bi-power"
                viewBox="0 0 16 16"
              >
                <path d="M7.5 1v7h1V1z" />
                <path d="M3 8.812a5 5 0 0 1 2.578-4.375l-.485-.874A6 6 0 1 0 11 3.616l-.501.865A5 5 0 1 1 3 8.812" />
              </svg>
            </button>
            <ul className="dropdown-menu">
              <li>
                <a className="dropdown-item" onClick={() => shutdownHost()}>
                  Shutdown
                </a>
              </li>
              <li>
                <a className="dropdown-item" onClick={() => restartHost()}>
                  Restart
                </a>
              </li>
            </ul>
          </div>
        </div>

        <h6 className="border-bottom" style={headerStyles}>
          Server Settings
        </h6>

        {isLoading ? (
          <p>Loading...</p>
        ) : error ? (
          <p>Error: {error.message}</p>
        ) : (
          <div>
            <DeviceSettings settings={data} saveSettings={saveSettings} />
          </div>
        )}
      </div>

      <ToastContainer
        className="p-3"
        position="bottom-center"
        style={{ zIndex: 1 }}
      >
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

function DeviceSettings({ settings, saveSettings }) {
  const [isFormDirty, setFormDirty] = useState(false);
  const [isTimezoneValid, setTimezoneValidity] = useState(true);
  const [isUsernameValid, setUsernameValidity] = useState(true);
  const [isPasswordValid, setPasswordValidity] = useState(true);
  const [autoUpdateApps, setAutoUpdateApps] = useState(true);
  const [autoUpdateOs, setAutoUpdateOs] = useState(true);
  const [username, setUsername] = useState('temp');
  const [password, setPassword] = useState('');
  const [timezone, setTimezone] = useState('America/Chicago');

  // effect runs on component mount
  useEffect(() => {
    setTimezone(settings.timezone);
    setAutoUpdateApps(settings.autoUpdateApps);
    setAutoUpdateOs(settings.autoUpdateOs);
    setUsername(settings.adminUser.username);
  }, [settings]);

  const handleSubmit = (e) => {
    e.preventDefault();
    const res = saveSettings({
      settings: {
        adminUser: {
          username: username,
          password: password,
        },
        timezone: timezone,
        autoUpdateApps: autoUpdateApps,
        autoUpdateOs: autoUpdateOs,
      },
    });
    // refetch();
  };

  const handleSetTimezone = (e) => {
    e.preventDefault();
    const tz = e.target.value;
    setFormDirty(true);
    setTimezone(tz);

    if (tz === 'NONE') {
      setTimezoneValidity(false);
    } else {
      setTimezoneValidity(true);
    }
  };

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
  };

  const handlePasswordChange = (e) => {
    e.preventDefault();
    const password = e.target.value;
    setFormDirty(true);
    setPassword(password);

    if (password.length > 0 && password.length < 4) {
      setPasswordValidity(false);
    } else {
      setPasswordValidity(true);
    }
  };

  return (
    <div className="tab-pane fade show active">
      <form className="row g-3">
        <div className="col-12">
          <select
            className={`form-select ${
              isFormDirty ? (isTimezoneValid ? 'is-valid' : 'is-invalid') : ''
            }`}
            value={timezone}
            onChange={(e) => handleSetTimezone(e)}
          >
            <option disabled value="NONE">
              Select a timezone...
            </option>
            <option value="America/New_York">Eastern (US)</option>
            <option value="America/Chicago">Central (US)</option>
            <option value="America/Denver">Mountain (US)</option>
            <option value="America/Los_Angeles">Pacific (US)</option>
          </select>
          <div
            className={`invalid-feedback ${
              isFormDirty ? (isTimezoneValid ? 'd-none' : '') : ''
            }`}
          >
            Please select a valid timezone
          </div>
        </div>

        <div className="col-12">
          <label className="form-label" htmlFor="usernameValidation">
            Username
          </label>
          <input
            id="usernameValidation"
            className={`form-control ${
              isFormDirty ? (isUsernameValid ? 'is-valid' : 'is-invalid') : ''
            }`}
            type="text"
            placeholder="Username"
            value={username}
            onChange={(e) => handleUsernameChange(e)}
            required
          />

          <div
            className={`invalid-feedback ${
              isFormDirty ? (isUsernameValid ? 'd-none' : '') : ''
            }`}
          >
            Username must be at least 4 characters long
          </div>
        </div>

        <div className="col-12">
          <input
            className={`form-control ${
              isFormDirty ? (isPasswordValid ? 'is-valid' : 'is-invalid') : ''
            }`}
            type="password"
            placeholder="Password (blank for no change)"
            value={password}
            onChange={(e) => handlePasswordChange(e)}
          />

          <div
            className={`invalid-feedback ${
              isFormDirty ? (isPasswordValid ? 'd-none' : '') : ''
            }`}
          >
            Password must be at least 4 characters long
          </div>
        </div>

        <div className="col-12">
          <div className="form-check form-switch form-check-reverse">
            <input
              className="form-check-input"
              type="checkbox"
              role="switch"
              value="autoUpdateApps"
              checked={autoUpdateApps ? true : false}
              onChange={() => setAutoUpdateApps(!autoUpdateApps)}
            />
            <label className="form-check-label">
              Automatically update applications
            </label>
          </div>
        </div>

        <div className="col-12">
          <div className="form-check form-switch form-check-reverse">
            <input
              className="form-check-input"
              type="checkbox"
              role="switch"
              value="autoUpdateOs"
              checked={autoUpdateOs ? true : false}
              onChange={() => setAutoUpdateOs(!autoUpdateOs)}
            />
            <label className="form-check-label">
              Automatically update server
            </label>
          </div>
        </div>

        <div className="col-12">
          <button
            style={{ float: 'right' }}
            className="btn btn-outline-primary"
            type="button"
            onClick={(e) => handleSubmit(e)}
          >
            Save
          </button>
        </div>
      </form>
    </div>
  );
}
