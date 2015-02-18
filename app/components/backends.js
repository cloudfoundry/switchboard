var Backend = require('./backend');
var React = require('react/addons');
var sortBy = require('lodash.sortby');
var types = React.PropTypes;

var Backends = React.createClass({
  propTypes: {
    backends: types.array.isRequired
  },

  renderBackends() {
    var {backends} = this.props;
    return backends && sortBy(backends, b => b.name).map(function(backend, i) {
      return (<Backend backend={backend} key={i}/>);
    });
  },

  render() {
    return (
      <table className="table table-data table-light man">
        <thead>
          <tr>
            <th className="col-sm-6">
              <h5 className="em-max mlm">Nodes</h5>
            </th>
            <th className="col-sm-6">
              <h5 className="em-max mlm">Status</h5>
            </th>
            <th className="col-sm-6">
              <h5 className="em-max mlm">Current Sessions</h5>
            </th>
            <th className="col-sm-6">
              <h5 className="em-max mlm">IP Address</h5>
            </th>
          </tr>
        </thead>
        <tbody>
        {this.renderBackends()}
        </tbody>
      </table>
    );
  }
});

module.exports = Backends;