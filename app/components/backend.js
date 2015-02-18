var React = require('react/addons');

var cx = React.addons.classSet;
var types = React.PropTypes;

var Backend = React.createClass({
  propTypes: {
    backend: types.object.isRequired
  },

  render() {
    var {name, healthy, currentSessionCount, host} = this.props.backend;
    const health = healthy ? 'HEALTHY' : 'UNHEALTHY';
    return (
      <tr>
        <td className="ptm txt-m">
          <h4 className="mlm">
            {name}
          </h4>
        </td>
        <td className="txt-m ptm">
          <h2>
            <span className={cx({'label label-primary mlm plm': true, 'bg-error-2': !healthy})}>
              <i className="fa fa-check fa-fw"></i>
            {health}
            </span>
          </h2>
        </td>
        <td className="txt-m">
          <h4 className="mlm">
          {currentSessionCount}
          </h4>
        </td>
        <td className="txt-m">
          <h4 className="mlm">
          {host}
          </h4>
        </td>
      </tr>
    );
  }
});

module.exports = Backend;