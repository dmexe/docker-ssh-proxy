import moment from 'moment'

import api from '../../../api/tasks'
import * as types from './types'

const state = {
  tasks: [],
  createdAt: null,
}

const getters = {
  getAll: state => state.tasks,
  getCreatedAt: state => state.created_at,
}

const actions = {
  [types.FETCH_ALL] (store) {
    api.getTasks().then((re) => {
      store.commit(types.FETCH_COMPLETE, {
        tasks: re.data.tasks,
        createdAt: moment(re.data.created_at),
      })
    })
  }
}

const mutations = {
  [types.FETCH_COMPLETE] (state, payload) {
    state.tasks = payload.tasks
    state.createdAt = payload.createdAt
  }
}

export default {
  namespaced: true,
  state: state,
  getters: getters,
  actions: actions,
  mutations: mutations,
}
