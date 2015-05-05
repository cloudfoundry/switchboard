global.React = require('react/addons');
var jQuery = require('jquery');
global.jQuery = jQuery;
global.$ = jQuery;

require('jasmine-ajax');
require('jasmine_dom_matchers');

beforeEach(function() {
  var Layout = require('../../serve/components/layout');
  spyOn(Layout, 'init');

  jasmine.clock().install();
  jasmine.Ajax.install();

  $('body').find('#root').remove().end().append('<div id="root"/>');
});

afterEach(function() {
  jasmine.Ajax.uninstall();
  jasmine.clock().uninstall();
});
