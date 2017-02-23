<template>
  <tr class="task">
    <td>
      <a class="task-instances button is-medium is-success">
        {{ task.instances.length }}
      </a>
    </td>

    <td>
      <a class="task-id button is-medium is-white">
        {{ task.id }}
      </a>
    </td>

    <td>
      <p class="task-image is-block">
        <label>image</label>
        <span>{{ task.image }}</span>
      </p>
      <p class="task-cpu is-block">
        <label>CPU</label>
        <span>{{ task.cpu }}</span>
      </p>
      <p class="task-mem is-block">
        <label>memory</label>
        <span>{{ task.mem * 1000 * 1000 | bytesSize }}</span>
      </p>
      <p class="task-updated-at is-block">
        <label>updated at</label>
        <span>{{ task.updatedAt | fromNow }}</span>
      </p>
    </td>
  </tr>
</template>

<style scoped>
  tr {
    border-top: 1px solid #eee;
  }

  td {
    padding: 1em 0;
  }

  label {
    display: inline-block;
    width: 6em;
    margin-right: .4em;
    text-align: right;
  }

  label:after {
    content: ":";
  }

  .task-id {
    margin-top: .3em;
  }

  .task-instances {
    margin-top: .3em;
    margin-right: .6em;
  }
</style>

<script src="./js/TaskTableRow.js"></script>
