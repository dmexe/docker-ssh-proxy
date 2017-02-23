import api from '../../../api/tasks'
import * as types from './types'

const state = {
  tasks:      [],
  createdAt:  null,
  lastError:  null,
  inProgress: false,
};

const getters = {
  getAll:       state => state.tasks,
  getCreatedAt: state => state.createdAt,
  getError:     state => state.lastError,
  isInProgress: state => state.inProgress,
};

const actions = {
  [types.FETCH_ALL] (store) {
    store.commit(types.FETCH_IN_PROGRESS);
    api
      .getTasks()
      .then((re) => {
        store.commit(types.FETCH_COMPLETE, re.data)
      })
      .catch((err) => {
        store.commit(types.FETCH_FAILED, err.message)
      });
  }
};

const mutations = {
  [types.FETCH_COMPLETE] (state, payload) {
    state.tasks      = payload.tasks;
    state.createdAt  = payload.createdAt;
    state.lastError  = null;
    state.inProgress = false;
  },

  [types.FETCH_FAILED] (state, err) {
    state.lastError  = err;
    state.inProgress = false;
  },

  [types.FETCH_IN_PROGRESS] (state) {
    state.inProgress = true;
  }
};

export default {
  namespaced: true,
  state: state,
  getters: getters,
  actions: actions,
  mutations: mutations,
};
