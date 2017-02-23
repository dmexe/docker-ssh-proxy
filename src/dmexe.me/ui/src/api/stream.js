import client from './client';

const baseURL = client.defaults.baseURL;
const streamURL = baseURL + "/v1/stream";

const addListener = (source) => {
  return (evType, cb) => {
    source.addEventListener(evType, cb)
  }
};

const subscribe = () => {
  const source = new EventSource(streamURL);

  return new Promise((resolve, reject) => {
    source.onopen = () => {
      console.debug("Open event stream at", streamURL);
      resolve(addListener(source));
    };

    source.onerror = (err) => {
      console.error("Stream error ",streamURL, err);
      reject(err);
    };
  })
};

export default subscribe;