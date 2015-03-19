var del = require('del');
var gulp = require('gulp');
var plugins = require('gulp-load-plugins')();
var express = require('express');
var lsof = require('lsof');
var portfinder = require('portfinder');
var runSequence = require('run-sequence');
var {spawn} = require('child_process');

gulp.task('clean-assets', function(callback) {
  del(['static/**/*'], callback);
});

gulp.task('assets', function(callback) {
  runSequence('clean-assets', ['serve-assets', 'app-assets'], callback);
});

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

function jasmineSpecs(options = {}) {
  return gulp.src('spec/app/**/*_spec.js')
    .pipe(plugins.cached('jasmine-javascript'))
    .pipe(plugins.webpack(Object.assign({
      devtool: 'eval',
      module: {
        loaders: [
          {test: /\.js$/, exclude: /node_modules/, loader: 'babel-loader?experimental=true'}
        ]
      },
      output: {
        filename: '[name].js'
      },
      quiet: true,
      watch: true
    }, options)));
}

gulp.task('spec', function(callback) {
  return jasmineSpecs({watch: false})
    .pipe(plugins.jasmineBrowser.specRunner({console: true}))
    .pipe(plugins.jasmineBrowser.phantomjs());
});


gulp.task('jasmine', function() {
  return jasmineSpecs()
    .pipe(plugins.jasmineBrowser.specRunner())
    .pipe(plugins.jasmineBrowser.server());
});

gulp.task('default', function(callback) {
  runSequence('lint', 'spec', callback);
});