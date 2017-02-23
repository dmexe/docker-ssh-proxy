// Note: You must restart bin/webpack-watcher for changes to take effect

const webpack = require('webpack')
const merge   = require('webpack-merge')
const config  = require('./shared.js')

module.exports = merge(config, {
  output: { filename: 'bundle.release.js' },

  plugins: [
    new webpack.LoaderOptionsPlugin({
      minimize: true
    }),
    new webpack.DefinePlugin({
      'process.env': {
        NODE_ENV: '"production"'
      }
    })
  ]
})

