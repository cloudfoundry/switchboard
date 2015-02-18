require('babel/polyfill');
var Backends = require('./backends');
var React = require('react/addons');
var Layout = require('../../serve/components/layout');
var request = require('superagent');

var Application = React.createClass({
  getInitialState() {
    return {backends: null}
  },

  componentDidMount() {
    request.get('/v0/backends')
      .accept('json')
      .end(function(err, {body: backends}) {
        if (err) return;
        this.setState({backends});
      }.bind(this));
  },

  render() {
    var {backends} = this.state;
    return (
      <div>
        <div className="main container-fluid pvm bg-dark-1">
          <div className="container pvn mvn">
            <div className="media">
              <a className="media-left" href="#">
                <img alt="Switchboard header logo" src="big-logo.png" width={35} height={35}/>
              </a>
              <div className="media-body">
                <h1 className="h3 man pts type-neutral-8">Pivotal Switchboard</h1>
              </div>
            </div>
          </div>
        </div>
        <div className="container">
          <div className="special">
            <h1 className="mbn title">CloudyApp</h1>
            <span className="i fa fa-refresh type-neutral-6 h5 man"></span>
            <span className="h5 mtn mls">Last Update: Just now</span>
            <hr className="divider-alternate-2 mtxl mbn"/>
            <div className="row man">
              <div className="alert alert-success bg-brand-4">
                <div className="media">
                  <div className="media-left">
                    <i className="fa fa-check-circle"></i>
                  </div>
                  <div className="media-body em-high">
                    All nodes are healthy!
                  </div>
                </div>
              </div>
            </div>
            <div className="row mtxl">
              <div className="col-sm-24 mtl">
                <div className="panel panel-alt bg-neutral-11 man">
                  <div className="panel-body pam">
                    <h1 className="mlm mts pan mbn type-neutral-3">Proxy0</h1>
                    <h5 className="mtn mlm mbl pan type-neutral-5">IP Address: 0.0.0.16</h5>
                  </div>
                </div>
                <hr className="divider-alternate-1 man"/>
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