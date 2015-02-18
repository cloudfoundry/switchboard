require('../spec_helper');

describe('Backends', function() {
  var Backend, subject, backends;
  beforeEach(function() {
    Backend = require('../../../app/components/backend');
    spyOn(Backend.type.prototype, 'render').and.callThrough();
    var Backends = require('../../../app/components/backends');
    backends = [
      {
        "host": "localhost",
        "port": 12345,
        "healthy": false,
        "active": false,
        "name": "backend - 2",
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
    subject = React.render(<Backends backends={backends}/>, root);
  });

  afterEach(function() {
    React.unmountComponentAtNode(root);
  });

  it('renders the table', function() {
    expect('table').toExist();
  });

  it('renders the backends', function() {
    expect(Backend.type.prototype.render).toHaveBeenCalled();
    expect(Backend.type.prototype.render.calls.count()).toEqual(backends.length);
  });

  it('orders the backends in sorted order by name', function() {
    expect($('tbody tr').map(function() { return $('td:eq(0)', this).text()}).toArray()).toEqual([
      'backend - 1',
      'backend - 2'
    ]);
  });
});