import * as React from 'react';

const loadingIcon = require('../assets/loading.gif');

export function SubmitButton({ text, loading, onClick }) {
  const callback = onClick ? true : false;
  return (
    <div className="col-12" display="inline-block">
      {callback && (
        <input
          style={{ float: 'right' }}
          className="btn btn-outline-primary"
          type="submit"
          value={text}
          disabled={loading}
          onClick={(e) => onClick(e)}
        />
      )}
      {!callback && (
        <input
          style={{ float: 'right' }}
          className="btn btn-outline-primary"
          type="submit"
          value={text}
          disabled={loading}
        />
      )}
      {loading && (
        <img
          src={loadingIcon}
          width="32"
          height="32"
          style={{ float: 'right', marginRight: '5px' }}
        />
      )}
    </div>
  );
}
