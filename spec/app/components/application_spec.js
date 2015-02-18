require('../spec_helper');

describe('Application', function() {
  var Backends, request, backends, subject;
  beforeEach(function() {
    backends = [
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
    var Application = require('../../../app/components/application');
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
  });
});