import Vue from 'vue'
import Vuex from 'vuex'
import logger from 'vuex/dist/logger'

import stream from './../api/stream';
import tasks from './modules/tasks'
import * as taskTypes from './modules/tasks/types'

Vue.use(Vuex)

const debug = process.env.NODE_ENV !== 'production'
const actions = {}
const getters = {}

const store = new Vuex.Store({
  actions,
  getters,
  modules: {
    tasks
  },
  strict: debug,
  plugins: debug ? [logger()] : []
});

const fetchTasks = (store) => store.dispatch('tasks/' + taskTypes.FETCH_ALL)

stream()
  .then((subscribe) => {
    subscribe("tasks.changed", () => fetchTasks(store))
  });

fetchTasks(store);

export default store
