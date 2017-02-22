import cli from './client'

export default {
  getTasks() {
    return cli.get('/v1/tasks')
  }
}
