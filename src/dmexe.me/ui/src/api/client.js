import axios from 'axios'

const client = axios.create({
  baseURL: 'http://localhost:2201/a'
});

export default client;
