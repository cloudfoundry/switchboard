require('../spec_helper');

describe('Application', function() {
  var Application, Backends, request, backends, subject;
  beforeEach(function() {
    backends = [
      {
        "host": "localhost",
        "port": 12345,
        "healthy": true,
        "active": false,
        "name": "backend - 1",
        "currentSessionCount": 0
      },
      {
        "host": "localhost",
        "port": 12345,
        "healthy": false,
        "active": false,
        "name": "backend - 1",
        "currentSessionCount": 0
      },
      {
        "host": "localhost",
        "port": 12345,
        "healthy": false,
        "active": false,
        "name": "backend - 1",
        "currentSessionCount": 0
      }
    ];
    Backends = require('../../../app/components/backends');
    spyOn(Backends.type.prototype, 'render').and.callThrough();
    Application = require('../../../app/components/application');
    subject = React.render(<Application/>, root);
    request = jasmine.Ajax.requests.mostRecent();
  });

  afterEach(function() {
    React.unmountComponentAtNode(root);
  });

  it('makes an ajax request', function() {
    expect(request).toBeDefined();
    expect(request.url).toEqual('/v0/backends');
  });


  describe('when some of the backends are unhealthy', function() {
    beforeEach(function() {
      subject.setState({backends});
    });

    it('renders the alert with the expected number of unhealthy nodes', function() {
      expect('.alert-error').toContainText('2 out of 3 nodes are unhealthy');
      expect($('.alert', root)).not.toHaveClass('bg-brand-4');
    });

    it('shows the fa-exclamation-triangle class', function() {
      expect('.fa-exclamation-triangle').toExist();
      expect('.fa-check-circle').not.toExist();
    });
  });

  describe('when all the backends are healthy', function() {
    beforeEach(function() {
      backends = backends.map(b => Object.assign({}, b, {healthy: true}));
      subject.setState({backends});
    });

    it('renders the alert with the all nodes are healthy', function() {
      expect('.alert-success').toContainText('All nodes are healthy!');
      expect($('.alert', root)).toHaveClass('bg-brand-4');
    });

    it('shows the fa-check-circle class', function() {
      expect('.fa-check-circle').toExist();
      expect('.fa-exclamation-triangle').not.toExist();
    });
  });

  describe('when the ajax request is successful', function() {
    beforeEach(function() {
      request.respondWith({
        status: 200,
        responseText: JSON.stringify(backends)
      });
    });

    it('renders the backends', function() {
      expect(Backends.type.prototype.render).toHaveBeenCalled();
    });

    describe('after some time has passed', function() {
      beforeEach(function() {
        jasmine.Ajax.requests.reset();
        jasmine.clock().tick(Application.POLL_INTERVAL);
        request = jasmine.Ajax.requests.mostRecent();
      });

      it('makes an ajax request', function() {
        expect(request).toBeDefined();
        expect(request.url).toEqual('/v0/backends');
      });
    });
  });
});
