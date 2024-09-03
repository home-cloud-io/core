import * as React from 'react';

import { 
    useShutdownHostMutation,
    useRestartHostMutation,
    useGetDeviceSettingsQuery,
} from '../../services/web_rpc';

export default function SettingsPage() { 
    const [shutdownHost, shutdownResult] = useShutdownHostMutation();
    const [restartHost, restartResult] = useRestartHostMutation();
    const { data, error, isLoading } = useGetDeviceSettingsQuery();

    const headerStyles = {
        paddingTop: ".75rem",
        paddingBottom: "1rem",
    }

    return (
        <div>
            <div className="my-3 p-3 bg-body rounded shadow-sm">
                <div className="float-end">
                
                    <div className="dropdown">
                    <button className="btn btn-secondary dropdown-toggle" type="button" data-bs-toggle="dropdown" aria-expanded="false">
                        <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" fill="currentColor" className="bi bi-power" viewBox="0 0 16 16">
                            <path d="M7.5 1v7h1V1z"/>
                            <path d="M3 8.812a5 5 0 0 1 2.578-4.375l-.485-.874A6 6 0 1 0 11 3.616l-.501.865A5 5 0 1 1 3 8.812"/>
                        </svg>
                    </button>
                    <ul className="dropdown-menu">
                        <li><a className="dropdown-item" onClick={() => shutdownHost()}>Shutdown</a></li>
                        <li><a className="dropdown-item" onClick={() => restartHost()}>Restart</a></li>
                    </ul>
                    </div>
                </div>

                <h6 className="border-bottom" style={headerStyles}>Server Settings</h6>

                {isLoading ? (
                    <p>Loading...</p>
                ) : error ? (
                    <p>Error: {error.message}</p>
                ) : (
                    <div>
                        <DeviceSettings settings={data}/>
                    </div>
                )}

            </div>
        </div>
    );
}

function DeviceSettings({settings}) {  
    const onChange = (e) => {
        e.preventDefault();
    }

    return (
      <div className="tab-pane fade show active"> 
        <form className="row g-3">
          <div className="col-12"> 
            <select
              className="form-select"
              value={settings.timezone}
              onChange={onChange}>
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
                checked={settings.autoUpdateApps} 
                onChange={onChange} />
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
                checked={settings.autoUpdateOs} 
                onChange={onChange}/>
              <label className="form-check-label">Automatically update server</label>
            </div>
          </div>
    
        </form>
      </div>
    );
  } 