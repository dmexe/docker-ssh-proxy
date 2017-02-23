<template>
  <div class="tasks">

    <p class="control">
      <input class="input is-large" type="text" autofocus placeholder="Type task name or image or label">
    </p>

    <div v-if="getError" class="notification is-warning">
      Could not load tasks: {{ getError }}
    </div>

    <TasksTable v-else :tasks="getAll"></TasksTable>
  </div>
</template>

<style scoped>
  .control {
    margin-bottom: 3rem;
  }
</style>

<script src="./js/Tasks.js"></script>

