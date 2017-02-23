import Vue from 'vue'
import Vuex from 'vuex'
import mapKeys from 'lodash.mapkeys'

import App from './components/App.vue'
import store from './store'
import filters from './filters'

Vue.use(Vuex);
mapKeys(filters, (v,k) => Vue.filter(k, v));

new Vue({
  el: '#main',
  store: store,
  render: h => h(App),
});
