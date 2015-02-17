var page = require('webpage').create();
var fs = require('fs');
var system = require('system');
var args = system.args;
var port = args[1] || 8888;

page.onConsoleMessage = function(message) {
  fs.write( '/dev/stdout', message, 'w');
};

page.onCallback = function(success) {
  phantom.exit(success ? 0 : 1);
};
page.open('http://localhost:' + port + '/console.html');
