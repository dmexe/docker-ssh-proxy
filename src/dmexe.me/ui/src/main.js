import Vue from 'vue'
import Vuex from 'vuex'
import moment from 'moment'

import App from './components/App.vue'
import store from './store'

Vue.use(Vuex)
Vue.filter("fromNow", function(str){
  return moment(str).fromNow()
})
Vue.filter("bytesSize", function(bytes){
  const si = true
  const thresh = si ? 1000 : 1024;
  if (Math.abs(bytes) < thresh) {
    return bytes + ' B';
  }
  const units = si
    ? ['kB','MB','GB','TB','PB','EB','ZB','YB']
    : ['KiB','MiB','GiB','TiB','PiB','EiB','ZiB','YiB'];
  var u = -1;

  do {
    bytes /= thresh;
    ++u;
  } while(Math.abs(bytes) >= thresh && u < units.length - 1);

  return bytes.toFixed(1)+' '+units[u];
})

new Vue({
  el: '#main',
  store: store,
  render: h => h(App),
})
