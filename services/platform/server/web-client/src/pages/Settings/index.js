import * as React from 'react';

import {
  useShutdownHostMutation,
  useRestartHostMutation,
  useGetDeviceSettingsQuery,
  useSetDeviceSettingsMutation,
  useEnableSecureTunnellingMutation,
  useDisableSecureTunnellingMutation,
  useRegisterToLocatorMutation,
  useDeregisterFromLocatorMutation,
} from '../../services/web_rpc';
import Button from 'react-bootstrap/Button';
import Dropdown from 'react-bootstrap/Dropdown';
import { useState, useEffect } from 'react';
import Toast from 'react-bootstrap/Toast';
import ToastContainer from 'react-bootstrap/ToastContainer';
import { SubmitButton } from '../../elements/buttons';

export default function SettingsPage() {
  const [shutdownHost, shutdownResult] = useShutdownHostMutation();
  const [restartHost, restartResult] = useRestartHostMutation();
  const [saveSettings, result] = useSetDeviceSettingsMutation();
  const [enableSecureTunnelling] = useEnableSecureTunnellingMutation();
  const [disableSecureTunnelling] = useDisableSecureTunnellingMutation();
  const [registerToLocator] = useRegisterToLocatorMutation();
  const [deregisterFromLocator] = useDeregisterFromLocatorMutation();
  const { data, error, isLoading } = useGetDeviceSettingsQuery();

  // TODO: make this way better
  if (isLoading) {
    return <div>Loading...</div>;
  }

  // TODO: make this way better
  if (error) {
    return <div>Error: {error.message}</div>;
  }

  return (
    <div>
      <div className="my-3 p-3 bg-body rounded shadow-sm">
        <div className="float-end">
          <div className="dropdown">
            <Dropdown>
              <Dropdown.Toggle variant="danger" id="dropdown-basic">
                Power
              </Dropdown.Toggle>
              <Dropdown.Menu>
                <Dropdown.Item onClick={() => shutdownHost()}>
                  Shutdown
                </Dropdown.Item>
                <Dropdown.Item onClick={() => restartHost()}>
                  Restart
                </Dropdown.Item>
              </Dropdown.Menu>
            </Dropdown>
          </div>
        </div>

        {isLoading ? (
          <p>Loading...</p>
        ) : error ? (
          <p>Error: {error.message}</p>
        ) : (
          <div>
            <h4 className="header border-bottom">
              <b>Server Settings</b>
            </h4>
            <DeviceSettings
              settings={data}
              saveSettings={saveSettings}
              disable={result.isLoading}
            />
            <h4 className="header border-bottom">
              <b>On the Go Settings</b>
            </h4>
            <OnTheGoSettings
              settings={data}
              enableSecureTunnelling={enableSecureTunnelling}
              disableSecureTunnelling={disableSecureTunnelling}
              registerToLocator={registerToLocator}
              deregisterFromLocator={deregisterFromLocator}
              disable={result.isLoading}
            />
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

function DeviceSettings({ settings, saveSettings, disable }) {
  const [isFormDirty, setFormDirty] = useState(false);
  const [isTimezoneValid, setTimezoneValidity] = useState(true);
  const [isUsernameValid, setUsernameValidity] = useState(true);
  const [isPasswordValid, setPasswordValidity] = useState(true);
  const [autoUpdateApps, setAutoUpdateApps] = useState(true);
  const [autoUpdateOs, setAutoUpdateOs] = useState(true);
  const [enableSsh, setEnableSsh] = useState(false);
  const [username, setUsername] = useState('temp');
  const [password, setPassword] = useState('');
  const [timezone, setTimezone] = useState('America/Chicago');
  const [sshKeys, setSshKeys] = useState([]);

  // effect runs on component mount
  useEffect(() => {
    setTimezone(settings.timezone);
    setAutoUpdateApps(settings.autoUpdateApps);
    setAutoUpdateOs(settings.autoUpdateOs);
    setUsername(settings.adminUser.username);
    setEnableSsh(settings.enableSsh);
    if (settings.trustedSshKeys) {
      setSshKeys(settings.trustedSshKeys);
    }
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
        enableSsh: enableSsh,
        trustedSshKeys: sshKeys,
      },
    });
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

  const handleSshKeyChange = (e) => {
    let keys = e.target.value.split('\n');
    if (keys.length === 1 && keys[0] === '') {
      keys = [];
    }
    setSshKeys(keys);
  };

  return (
    <div className="tab-pane fade show active">
      <form className="row g-3">
        <div className="col-12">
          <label className="form-label">Timezone</label>
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
            User
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
          <label className="form-label">Updates</label>
          <div className="form-check form-switch">
            <label className="form-check-label">
              Automatically update applications
            </label>
            <input
              className="form-check-input"
              type="checkbox"
              role="switch"
              value="autoUpdateApps"
              checked={autoUpdateApps ? true : false}
              onChange={() => setAutoUpdateApps(!autoUpdateApps)}
            />
          </div>
          <div className="form-check form-switch">
            <label className="form-check-label">
              Automatically update server
            </label>
            <input
              className="form-check-input"
              type="checkbox"
              role="switch"
              value="autoUpdateOs"
              checked={autoUpdateOs ? true : false}
              onChange={() => setAutoUpdateOs(!autoUpdateOs)}
            />
          </div>
        </div>

        <div className="col-12">
          <label className="form-label">SSH</label>
          <div className="form-check form-switch">
            <label className="form-check-label">Enable SSH access</label>
            <input
              className="form-check-input"
              type="checkbox"
              role="switch"
              value="enableSsh"
              checked={enableSsh ? true : false}
              onChange={() => setEnableSsh(!enableSsh)}
            />
          </div>
          {enableSsh && (
            <div>
              <label className="form-check-label">
                Input trusted SSH keys one per line (optional)
              </label>
              <textarea
                className="form-textarea textarea-fill"
                value={sshKeys.join('\n')}
                onChange={(e) => handleSshKeyChange(e)}
              />
            </div>
          )}
        </div>

        <SubmitButton text="Save" loading={disable} onClick={handleSubmit} />
      </form>
    </div>
  );
}

function OnTheGoSettings({
  settings,
  enableSecureTunnelling,
  disableSecureTunnelling,
  registerToLocator,
  deregisterFromLocator,
  disable,
}) {
  const [enableOnTheGo, setEnableOnTheGo] = useState(false);
  const [locatorToAdd, setlocatorToAdd] = useState(
    'https://locator.home-cloud.io'
  );
  const [locators, setLocators] = useState([]);

  // effect runs on component mount
  useEffect(() => {
    if (settings.locatorSettings) {
      setEnableOnTheGo(settings.locatorSettings.enabled);
      if (settings.locatorSettings.locators) {
        setLocators(
          Object.keys(settings.locatorSettings.locators).map(
            (k) => settings.locatorSettings.locators[k]
          )
        );
      }
    }
  }, [settings]);

  const handleEnable = () => {
    enableSecureTunnelling();
    setEnableOnTheGo(true);
  };

  const handleDisable = () => {
    disableSecureTunnelling();
    setEnableOnTheGo(false);
    setLocators([]);
  };

  const handleRegister = async () => {
    const { data, error } = await registerToLocator({
      locatorAddress: locatorToAdd,
    });
    if (error) {
      console.error(error);
    } else {
      setLocators((locators) => [
        ...locators,
        {
          address: locatorToAdd,
          serverId: data.serverId,
          wireguardInterface: 'wg0',
        },
      ]);
    }
  };

  const handleDeregister = ({ locator }) => {
    setLocators(
      locators.filter(function (l) {
        return l.address != locator.address || l.serverId != locator.serverId;
      })
    );
    deregisterFromLocator({
      locatorAddress: locator.address,
      serverId: locator.serverId,
    });
  };

  const handleLocatorChange = (e) => {
    setlocatorToAdd(e.target.value);
  };

  return (
    <div>
      <div>
        <div>
          {!enableOnTheGo && (
            <div className="d-flex pt-4">
              <Button onClick={() => handleEnable()}>Enable</Button>
            </div>
          )}
          {enableOnTheGo && (
            <div>
              <div className="d-flex pt-4">
                <Button variant="warning" onClick={() => handleDisable()}>
                  Disable
                </Button>
              </div>
              <div className="d-flex pt-4">
                <label className="form-check-label">
                  Add Locator servers:
                  <div className="d-flex">
                    <input
                      style={{ width: '300px' }}
                      type="text"
                      placeholder="Locator address"
                      value={locatorToAdd}
                      onChange={(e) => handleLocatorChange(e)}
                      required
                    />
                    <Button variant="success" onClick={() => handleRegister()}>
                      Add
                    </Button>
                  </div>
                </label>
              </div>
            </div>
          )}
        </div>

        <div className="card">
          {enableOnTheGo && (
            <div className="border-bottom">Locator Servers:</div>
          )}
          {enableOnTheGo &&
            locators.map((locator, index) => {
              return (
                <LocatorCard
                  key={index}
                  locator={locator}
                  handle={handleDeregister}
                />
              );
            })}
        </div>
      </div>
    </div>
  );
}

function LocatorCard({ locator, handle }) {
  const rowStyles = {
    paddingLeft: '2rem',
  };

  return (
    <div className="d-flex text-body-secondary pt-4">
      <div
        className="pb-3 mb-0 small lh-sm border-bottom w-100 position-relative"
        style={rowStyles}
      >
        <div className="d-flex justify-content-between">
          <div className="text-gray-dark">
            <strong>Server: </strong> {locator.address}
            <br />
            <strong>Identifier: </strong> {locator.serverId}
          </div>
          <Button variant="danger" onClick={() => handle({ locator })}>
            Remove
          </Button>
        </div>
      </div>
    </div>
  );
}
