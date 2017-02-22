<template>
  <div class="container">
    <div class="section">
      <p class="control">
        <input class="input is-large" type="text" autofocus placeholder="Type task name or image or label">
      </p>

      <table class="tasks-table">
        <thead>
          <th>Tasks</th>
          <th>Name</th>
          <th>Info</th>
        </thead>

        <tr v-for="task in getAll" class="task-row">
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
              <span>{{ task.updated_at | fromNow }}</span>
            </p>
          </td>
        </tr>
      </table>
    </div>
  </div>
</template>

<style src="bulma/css/bulma.css"></style>
<style src="./app.css"></style>
<script src="./app.js"></script>
