import * as React from 'react';
import { useDispatch, shallowEqual, useSelector } from 'react-redux';
import { FileUploadStatus, setFileUploadStatus } from '../../services/web_slice';


const loading = require('../../assets/loading.gif');

let BASE_URL = '';
let LOCAL_DOMAIN = 'localhost';

if (process.env.NODE_ENV === 'development') {
  BASE_URL = `http://${LOCAL_DOMAIN}:8000`;
} else {
  BASE_URL = 'http://home-cloud.local';
}

export default function UploadPage() {
  const uploadStatus = useSelector(state => state.server.file_upload_status, shallowEqual);

  const headerStyles = {
    paddingTop: '.75rem',
    paddingBottom: '1rem',
  };

  return (
    <div>
      <div className="my-3 p-3 bg-body rounded shadow-sm">
        <h6 className="border-bottom" style={headerStyles}>
          Upload File
        </h6>
        <div>
          <UploadForm
            status={uploadStatus["file_id"]} />
        </div>
      </div>
    </div>
  );
}


function UploadForm({status = FileUploadStatus.DEFAULT}) {
  const dispatch = useDispatch();

  const handleSubmit = (e) => {
    status = FileUploadStatus.UPLOADING;
    let id = "file_id"
    dispatch(setFileUploadStatus({status, id}));
  };

  return (
    <div className="tab-pane fade show active">
      <iframe name="dummyframe" id="dummyframe" style={{ display: 'none' }} ></iframe>
      <form
        className="row g-3"
        action={`${BASE_URL}/upload`}
        method="post"
        encType="multipart/form-data"
        target="dummyframe"
        onSubmit={(e) => handleSubmit(e)}
      >
        <div className="col-12">
          <div className="container" display="inline-block" title="Choose the installed App you want to upload this file to. For example, you could upload a movie file to your Jellyfin collection." >
            <svg version="1.1" id="_x32_" xmlns="http://www.w3.org/2000/svg" xmlnsXlink="http://www.w3.org/1999/xlink" viewBox="0 0 512 512" xmlSpace="preserve" width="20" >
              <g>
                <path className="st0" d="M306.068,156.129c-6.566-5.771-14.205-10.186-22.912-13.244c-8.715-3.051-17.82-4.58-27.326-4.58   c-9.961,0-19.236,1.59-27.834,4.752c-8.605,3.171-16.127,7.638-22.576,13.41c-6.449,5.772-11.539,12.9-15.272,21.384   c-3.736,8.486-5.604,17.937-5.604,28.34h44.131c0-7.915,2.258-14.593,6.785-20.028c4.524-5.426,11.314-8.138,20.369-8.138   c8.598,0,15.328,2.661,20.197,7.974c4.864,5.322,7.297,11.942,7.297,19.856c0,3.854-0.965,7.698-2.887,11.543   c-1.922,3.854-4.242,7.586-6.959,11.197l-21.26,27.232c-4.527,5.884-16.758,22.908-16.758,40.316v10.187h44.129v-7.128   c0-2.938,0.562-5.996,1.699-9.168c1.127-3.162,6.453-10.904,8.268-13.168l21.264-28.243c4.752-6.333,8.705-12.839,11.881-19.518   c3.166-6.67,4.752-14.308,4.752-22.913c0-10.86-1.926-20.478-5.772-28.85C317.832,168.969,312.627,161.892,306.068,156.129z" />
                <rect x="234.106" y="328.551" className="st0" width="46.842" height="45.144" />
                <path className="st0" d="M256,0C114.613,0,0,114.615,0,256s114.613,256,256,256c141.383,0,256-114.615,256-256S397.383,0,256,0z    M256,448c-105.871,0-192-86.131-192-192S150.129,64,256,64c105.867,0,192,86.131,192,192S361.867,448,256,448z" />
              </g>
            </svg>

            <label className="form-label" >
              &ensp;Select an App:
            </label>
          </div>
          <select className="form-select" id="app" name="app" defaultValue="" required >
            <option hidden disabled value=""> -- select an option -- </option>
            <option value="jellyfin">Jellyfin</option>
            <option value="immich">Immich</option>
          </select>
        </div>

        <div className="col-12">
          <div className="container" display="inline-block" title="Input the path within the selected App's storage to upload the file. For example, if you have a folder called 'movies/family/' in your Jellyfin collection you would type 'movies/family/' here." >
            <svg version="1.1" id="_x32_" xmlns="http://www.w3.org/2000/svg" xmlnsXlink="http://www.w3.org/1999/xlink" viewBox="0 0 512 512" xmlSpace="preserve" width="20" >
              <g>
                <path className="st0" d="M306.068,156.129c-6.566-5.771-14.205-10.186-22.912-13.244c-8.715-3.051-17.82-4.58-27.326-4.58   c-9.961,0-19.236,1.59-27.834,4.752c-8.605,3.171-16.127,7.638-22.576,13.41c-6.449,5.772-11.539,12.9-15.272,21.384   c-3.736,8.486-5.604,17.937-5.604,28.34h44.131c0-7.915,2.258-14.593,6.785-20.028c4.524-5.426,11.314-8.138,20.369-8.138   c8.598,0,15.328,2.661,20.197,7.974c4.864,5.322,7.297,11.942,7.297,19.856c0,3.854-0.965,7.698-2.887,11.543   c-1.922,3.854-4.242,7.586-6.959,11.197l-21.26,27.232c-4.527,5.884-16.758,22.908-16.758,40.316v10.187h44.129v-7.128   c0-2.938,0.562-5.996,1.699-9.168c1.127-3.162,6.453-10.904,8.268-13.168l21.264-28.243c4.752-6.333,8.705-12.839,11.881-19.518   c3.166-6.67,4.752-14.308,4.752-22.913c0-10.86-1.926-20.478-5.772-28.85C317.832,168.969,312.627,161.892,306.068,156.129z" />
                <rect x="234.106" y="328.551" className="st0" width="46.842" height="45.144" />
                <path className="st0" d="M256,0C114.613,0,0,114.615,0,256s114.613,256,256,256c141.383,0,256-114.615,256-256S397.383,0,256,0z    M256,448c-105.871,0-192-86.131-192-192S150.129,64,256,64c105.867,0,192,86.131,192,192S361.867,448,256,448z" />
              </g>
            </svg>

            <label className="form-label" >
              &ensp;File path (optional):
            </label>
          </div>
          <input
            className="form-control"
            id="path"
            name="path"
            type="text"
          />
        </div>

        <div className="col-12">
          <div className="container" display="inline-block" title="You can optionally change the file's name during upload by inputing a new name here. For example, if you want the file to be called 'example.mov' simply type that here." >
            <svg version="1.1" id="_x32_" xmlns="http://www.w3.org/2000/svg" xmlnsXlink="http://www.w3.org/1999/xlink" viewBox="0 0 512 512" xmlSpace="preserve" width="20" >
              <g>
                <path className="st0" d="M306.068,156.129c-6.566-5.771-14.205-10.186-22.912-13.244c-8.715-3.051-17.82-4.58-27.326-4.58   c-9.961,0-19.236,1.59-27.834,4.752c-8.605,3.171-16.127,7.638-22.576,13.41c-6.449,5.772-11.539,12.9-15.272,21.384   c-3.736,8.486-5.604,17.937-5.604,28.34h44.131c0-7.915,2.258-14.593,6.785-20.028c4.524-5.426,11.314-8.138,20.369-8.138   c8.598,0,15.328,2.661,20.197,7.974c4.864,5.322,7.297,11.942,7.297,19.856c0,3.854-0.965,7.698-2.887,11.543   c-1.922,3.854-4.242,7.586-6.959,11.197l-21.26,27.232c-4.527,5.884-16.758,22.908-16.758,40.316v10.187h44.129v-7.128   c0-2.938,0.562-5.996,1.699-9.168c1.127-3.162,6.453-10.904,8.268-13.168l21.264-28.243c4.752-6.333,8.705-12.839,11.881-19.518   c3.166-6.67,4.752-14.308,4.752-22.913c0-10.86-1.926-20.478-5.772-28.85C317.832,168.969,312.627,161.892,306.068,156.129z" />
                <rect x="234.106" y="328.551" className="st0" width="46.842" height="45.144" />
                <path className="st0" d="M256,0C114.613,0,0,114.615,0,256s114.613,256,256,256c141.383,0,256-114.615,256-256S397.383,0,256,0z    M256,448c-105.871,0-192-86.131-192-192S150.129,64,256,64c105.867,0,192,86.131,192,192S361.867,448,256,448z" />
              </g>
            </svg>

            <label className="form-label" >
              &ensp;Change file name on upload (optional):
            </label>
          </div>
          <input
            className="form-control"
            id="file-name-override"
            name="file-name-override"
            type="text"
          />
        </div>

        <div className="col-12">
          <div className="container" display="inline-block" title="Choose the file to upload. This can be any file you want: videos, music, photos, etc." >
            <svg version="1.1" id="_x32_" xmlns="http://www.w3.org/2000/svg" xmlnsXlink="http://www.w3.org/1999/xlink" viewBox="0 0 512 512" xmlSpace="preserve" width="20" >
              <g>
                <path className="st0" d="M306.068,156.129c-6.566-5.771-14.205-10.186-22.912-13.244c-8.715-3.051-17.82-4.58-27.326-4.58   c-9.961,0-19.236,1.59-27.834,4.752c-8.605,3.171-16.127,7.638-22.576,13.41c-6.449,5.772-11.539,12.9-15.272,21.384   c-3.736,8.486-5.604,17.937-5.604,28.34h44.131c0-7.915,2.258-14.593,6.785-20.028c4.524-5.426,11.314-8.138,20.369-8.138   c8.598,0,15.328,2.661,20.197,7.974c4.864,5.322,7.297,11.942,7.297,19.856c0,3.854-0.965,7.698-2.887,11.543   c-1.922,3.854-4.242,7.586-6.959,11.197l-21.26,27.232c-4.527,5.884-16.758,22.908-16.758,40.316v10.187h44.129v-7.128   c0-2.938,0.562-5.996,1.699-9.168c1.127-3.162,6.453-10.904,8.268-13.168l21.264-28.243c4.752-6.333,8.705-12.839,11.881-19.518   c3.166-6.67,4.752-14.308,4.752-22.913c0-10.86-1.926-20.478-5.772-28.85C317.832,168.969,312.627,161.892,306.068,156.129z" />
                <rect x="234.106" y="328.551" className="st0" width="46.842" height="45.144" />
                <path className="st0" d="M256,0C114.613,0,0,114.615,0,256s114.613,256,256,256c141.383,0,256-114.615,256-256S397.383,0,256,0z    M256,448c-105.871,0-192-86.131-192-192S150.129,64,256,64c105.867,0,192,86.131,192,192S361.867,448,256,448z" />
              </g>
            </svg>

            <label className="form-label" >
              &ensp;Select a file:
            </label>
          </div>
          <input
            id="file"
            className="form-control"
            type="file"
            name="file"
            required
          />
        </div>

        <div className="container" display="inline-block" >
          { status === FileUploadStatus.UPLOADING ? <LoadingIcon/> : <SubmitButton/> }
        </div>
      </form>
    </div>
  );
}

const LoadingIcon = () => {
  return (
    <img
      src={loading}
      width="32"
      height="32"
      style={{ float: 'right' }}
    />
  );
}

const SubmitButton = () => {
  return (
    <div className="col-12">
      <input
        style={{ float: 'right' }}
        className="btn btn-outline-primary"
        type="submit"
        value="Upload"
      />
    </div>
  );
}