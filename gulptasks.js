var gulp = require('gulp');
var plugins = require('gulp-load-plugins')();
var express = require('express');
var lsof = require('lsof');
var portfinder = require('portfinder');
var runSequence = require('run-sequence');
var {spawn} = require('child_process');

gulp.task('assets', ['serve-assets', 'app-assets']);

gulp.task('serve-assets', function() {
  var serve = require('./serve/serve');
  return gulp.src('serve/**/*.js')
    .pipe(plugins.file('index.html', serve()))
    .pipe(plugins.filter('index.html'))
    .pipe(gulp.dest('static'));
});


gulp.task('app-assets-vendor', function() {
  return gulp.src('vendor/pui-v1.4.0/**/*')
    .pipe(plugins.copy('static', {prefix: 2}));
});

gulp.task('app-assets-images', function() {
  return gulp.src('app/images/*')
    .pipe(plugins.copy('static', {prefix: 2}));
});

gulp.task('app-assets-stylesheets', function() {
  return gulp.src('app/stylesheets/**/*.styl')
    .pipe(plugins.stylus())
    .pipe(gulp.dest('static'));
});

gulp.task('app-assets-build', function() {
  return gulp.src('app/**/*.js')
    .pipe(plugins.webpack({
      devtool: 'source-map',
      entry: {
        application: `./app/components/application.js`
      },
      module: {
        loaders: [
          {test: /\.js$/, exclude: /node_modules/, loader: 'babel-loader?experimental=true'}
        ]
      },
      output: {
        filename: '[name].js'
      }
    }))
    .pipe(gulp.dest('static'));
});

gulp.task('app-assets', ['app-assets-vendor', 'app-assets-build', 'app-assets-stylesheets', 'app-assets-images']);

gulp.task('watch-serve-assets', function() {
  gulp.watch(['serve/**/*.js', 'app/**/*.js'], ['serve-assets']);
});

gulp.task('watch-app-assets', function() {
  gulp.watch(['app/**/*.js', 'app/stylesheets/**/*.styl', 'app/images/**/*'], ['app-assets']);
});

gulp.task('watch-assets', ['watch-serve-assets', 'watch-app-assets']);

gulp.task('s', ['assets', 'watch-assets']);

gulp.task('lint', function () {
  return gulp.src(['app/**/*.js'])
    .pipe(plugins.babel())
    .pipe(plugins.jshint())
    .pipe(plugins.jshint.reporter('jshint-stylish'))
    .pipe(plugins.jshint.reporter('fail'));
});

gulp.task('spec', function(callback) {
  runSequence('lint', 'spec', callback);
});

function jasmineConsoleReporter(port, callback) {
  var phantomjs = spawn('phantomjs', ['spec/support/console_reporter.js', port], {stdio: 'inherit', env: process.env});
  phantomjs.on('close', callback);
}

gulp.task('spec', function(callback) {
  var port = 8888;
  lsof.rawTcpPort(port, function(data) {
    if (data.length) {
      jasmineConsoleReporter(port, callback);
    } else {
      portfinder.getPort(function(err, port) {
        if (err) return callback(err);
        var env = Object.assign({}, process.env, {JASMINE_PORT: port});
        var server = spawn('./node_modules/.bin/gulp', ['jasmine'], {env});
        server.stdout.on('data', function(data) {
          var output = data.toString();
          if (output.includes('Jasmine server listening on ')) {
            jasmineConsoleReporter(port, function(err) {
              process.kill(server.pid, 'SIGINT');
              callback(err);
            });
          }
        });
      });
    }
  });
});

gulp.task('jasmine-assets', function() {
  return gulp.src(['spec/spec.js', 'spec/app/**/*_spec.js'])
    .pipe(plugins.cached('jasmine-javascript'))
    .pipe(plugins.webpack({
      devtool: 'eval',
      entry: {
        spec: `./spec/spec.js`
      },
      module: {
        loaders: [
          {test: /\.js$/, exclude: /node_modules/, loader: 'babel-loader?experimental=true'}
        ]
      },
      output: {
        filename: '[name].js'
      },
      watch: true

    }))
    .pipe(gulp.dest('tmp/jasmine'));
});

gulp.task('jasmine-server', function() {
  function createServer({port, onReady}) {
    var app = express();
    app.use(express.static(__dirname + '/tmp/jasmine'));
    app.use(express.static(__dirname + '/spec/app/public'));
    app.listen(port, onReady && function() { onReady(port); });
    return app;
  }
  var port = process.env.JASMINE_PORT || 8888;
  createServer({port, onReady: port => plugins.util.log(`Jasmine server listening on ${port}`)});
});

gulp.task('jasmine', ['jasmine-assets', 'jasmine-server']);