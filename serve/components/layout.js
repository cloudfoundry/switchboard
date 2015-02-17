var React = require('react');

var types = React.PropTypes;

var Body = React.createClass({
  propTypes: {
    entry: types.func.isRequired,
    scripts: types.array.isRequired
  },

  render() {
    var {config, entry, scripts, className} = this.props;
    scripts = scripts.map(function(src, i) {
      return (<script type="text/javascript" src={src} key={i}/>);
    });
    var Entry = React.createFactory(entry);
    var __html = React.renderToString(Entry({config}));
    var configScript = `var switchboard = {}; switchboard.config = ${JSON.stringify(config)};`;
    return (
      <body className={className}>
        <div id="root" dangerouslySetInnerHTML={{__html}}/>
        <script type="text/javascript" dangerouslySetInnerHTML={{__html: configScript}}/>
        {scripts}
      </body>
    );
  }
});

var Layout = React.createClass({
  propTypes: {
    entry: types.func.isRequired,
    stylesheets: types.array.isRequired,
    scripts: types.array.isRequired,
    config: types.object
  },

  statics: {
    init(Entry) {
      if (typeof document !== 'undefined') {
        React.render(<Entry {...{config: switchboard.config}}/>, root);
      }
    }
  },

  render() {
    var {stylesheets} = this.props;

    stylesheets = stylesheets.map(function(href, i) {
      return (<link rel="stylesheet" type="text/css" href={href} key={i}/>);
    });

    return (
      <html>
        <head>{stylesheets}</head>
        <Body {...this.props}/>
      </html>
    );
  }
});

module.exports = Layout;