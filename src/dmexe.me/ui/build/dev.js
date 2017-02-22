// Note: You must restart bin/webpack-watcher for changes to take effect

const webpack = require('webpack')
const merge   = require('webpack-merge')
const path    = require('path')
const config  = require('./shared.js')

module.exports = merge(config, {
  devtool: 'sourcemap',

  devServer: {
    contentBase: path.join(__dirname, '..'),
    compress: true,
    port: 9000
  },

  stats: {
    errorDetails: true
  },

  output: {
    pathinfo: true
  },

  plugins: [
    new webpack.LoaderOptionsPlugin({
      debug: true
    })
  ]
})
