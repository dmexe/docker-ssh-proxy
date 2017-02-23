import { mapGetters } from 'vuex'
import TasksTable from '../TasksTable.vue'

export default {
  name: "Tasks",
  components: { TasksTable },
  computed: {
    ...mapGetters('tasks', ['getAll', 'getError'])
  }
}