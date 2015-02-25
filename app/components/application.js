require('babel/polyfill');
var Backends = require('./backends');
var React = require('react/addons');
var Layout = require('../../serve/components/layout');
var request = require('superagent');
var {setCorrectingInterval} = require('correcting-interval');

var cx = React.addons.classSet;

var Application = React.createClass({
  getInitialState() {
    return {backends: []}
  },

  statics: {
    POLL_INTERVAL: 1 * 1000
  },

  componentDidMount() {
    this.pollBackends();
  },

  updateBackends() {
    request.get('/v0/backends')
      .accept('json')
      .end(function(err, {body: backends}) {
        if (err) return;
        this.setState({backends});
      }.bind(this));
  },

  pollBackends() {
    this.updateBackends();
    setCorrectingInterval(this.updateBackends, Application.POLL_INTERVAL);
  },

  render() {
    var {backends} = this.state;
    var healthyCount = backends && backends.reduce((memo, b) => memo + (b.healthy ? 1 : 0), 0);
    var healthy = healthyCount === backends.length;
    var healthText = healthy ? 'All nodes are healthy!' : `${backends.length - healthyCount} out of ${backends.length} nodes are unhealthy.`;
    return (
      <div>
        <div className="main container-fluid pvm bg-dark-1">
          <div className="container pvn mvn">
            <div className="media">
              <a className="media-left" href="#">
                <img alt="Switchboard header logo" src="switchboard-header-logo.png" width={40} height={40}/>
              </a>
              <div className="media-body">
                <h1 className="h3 man pts type-neutral-8">Switchboard</h1>
              </div>
            </div>
          </div>
        </div>
        <div className="container">
          <div className="special">
            <div className="row man">
              <div className={cx({'alert': true, 'alert-success bg-brand-4': healthy, 'alert-error': !healthy})}>
                <div className="media">
                  <div className="media-left media-middle">
                    <i className={cx({'fa': true, 'fa-check-circle': healthy, 'fa-exclamation-triangle': !healthy})}></i>
                  </div>
                  <div className="media-body em-high">
                  {healthText}
                  </div>
                </div>
              </div>
            </div>
            <div className="row mtxl">
              <div className="col-sm-24 mtl">
                <Backends backends={backends}/>
              </div>
            </div>
          </div>
        </div>
      </div>
    );
  }
});

if (typeof window !== 'undefined') {
  Layout.init(Application);
}

module.exports = Application;
