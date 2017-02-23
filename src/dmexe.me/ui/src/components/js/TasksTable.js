import TaskTableRow from '../TaskTableRow.vue'

export default {
  name: "TasksTable",
  components: { TaskTableRow },
  props: {
    tasks: Array,
  }
}
