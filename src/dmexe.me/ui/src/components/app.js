import { mapGetters } from 'vuex'

export default {
  name: "App",
  computed: {
    ...mapGetters('tasks', ['getAll'])
  }
}
