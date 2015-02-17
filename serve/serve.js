var React = require('react/addons');
var Layout = require('./components/layout');

module.exports = function() {
  var Application = require('../app/components/application');
  var stylesheets = ['pivotal-ui.min.css', 'application.css'];
  var scripts = ['application.js'];
  var config = {};
  var className = 'bg-neutral-9';
  var props = {entry: Application, stylesheets, scripts, config, className};
  return React.renderToStaticMarkup(<Layout {...props}/>);
};