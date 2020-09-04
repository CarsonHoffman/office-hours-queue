const path = require('path');
var webpack = require('webpack');
const CompressionPlugin = require('compression-webpack-plugin');

module.exports = {
  // mode: "development",
  entry: './src/js/main.ts',
  devtool: "inline-source-map",
  output: {
    path: path.join(__dirname, '/public/js/'),
    filename: 'index_bundle.js',
    libraryTarget: 'var',
    library: 'Bundle'
  },
  module: {
    rules: [
      {
        test: /\.tsx?$/,
        use: 'ts-loader',
        exclude: /node_modules/
      },
      {
        test: /\.css$/,
        use: ['style-loader', 'css-loader']
      },
      {
        test: /\.(woff(2)?|ttf|eot|svg)(\?v=\d+\.\d+\.\d+)?$/,
        use: [
          {
            loader: 'file-loader',
            options: {
              name: '[name].[ext]',
              outputPath: 'fonts/'
            }
          }
        ]
      },
      { 
        test: /bootstrap.+\.(jsx|js)$/,
        loader: 'imports-loader?jQuery=jquery,$=jquery,this=>window'
      }
    ],
  },
  resolve: {
    extensions: [ '.tsx', '.ts', '.js' ]
  },
  plugins: [
    new webpack.ContextReplacementPlugin(/moment[\\\/]locale$/, /^\.\/(en|zh-tw)$/),
    new CompressionPlugin(),
  ]
};
