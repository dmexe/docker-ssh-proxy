<template>
  <table>
    <thead>
      <th>Tasks</th>
      <th>Name</th>
      <th>Info</th>
    </thead>

    <TaskTableRow v-for="task in tasks" :task="task"></TaskTableRow>
  </table>
</template>

<style scoped>
  th {
    font-weight: normal;
    padding-bottom: 1em;
    font-size: 1rem;
  }

  th:nth-child(2) {
    text-align: center;
  }

  th:nth-child(3) {
    padding-left: 6.6em;
  }
</style>

<script src="./js/TasksTable.js"></script>
