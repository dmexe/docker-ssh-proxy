// Note: You must restart bin/webpack-watcher for changes to take effect

const path = require('path')
const glob = require('glob')
const extname = require('path-complete-extname')

module.exports = {
  entry: path.resolve(__dirname, "../src/main.js"),

  output: {
    filename: 'bundle.js',
    path: path.resolve(__dirname, '../dist')
  },

  module: {
    rules: [
      { test: /\.(woff2?|ttf|eot|svg)$/, loader: "url-loader" },
      { test: /\.png$/, loader: "file-loader" },
      { test: /\.css$/, loader: "style-loader!css-loader" },
      {
        test: /\.js$/,
        exclude: /node_modules/,
        loader: 'babel-loader'
      },
      {
        test: /\.vue$/,
        loader: 'vue-loader'
      }
    ]
  },

  plugins: [],

  resolve: {
    extensions: [ '.js', '.css' ],
    modules: [
      path.resolve(__dirname, '../node_modules')
    ],
    alias: {
      'vue$': 'vue/dist/vue.common.js'
    }
  },

  resolveLoader: {
    modules: [ path.resolve(__dirname, '../node_modules') ]
  }
}

