require('../spec_helper');

describe('Backend', function() {
  var subject, backend;
  beforeEach(function() {
    var Backend = require('../../../app/components/backend');
    backend = {
      "host": "localhost",
      "port": 12345,
      "healthy": false,
      "active": false,
      "name": "backend - 1",
      "currentSessionCount": 0
    };
    $(root).append('<table><tbody/></table>');
    subject = React.render(<Backend backend={backend}/>, $('tbody')[0]);
  });

  afterEach(function() {
    React.unmountComponentAtNode(root);
  });

  it('renders a backend', function() {
    expect('tr').toExist();
  });

  it('renders the expected metadata on a backend', function() {
    var {name, healthy, currentSessionCount, host} = backend;
    const health = healthy ? 'HEALTHY' : 'UNHEALTHY';
    expect($('td').map(function() {
      return $(this).text().trim();
    }).toArray()).toEqual([
      name,
      health,
      currentSessionCount.toString(),
      host
    ]);
  });

  describe('when the backend is not healthy', function() {
    beforeEach(function() {
      backend.healthy = false;
      subject.setProps({backend});
    });

    it('renders unhealthy', function() {
      expect('tr').toContainText('UNHEALTHY');
    });

    it('adds the bg-error-2 class', function() {
      expect('.bg-error-2').toExist();
    });

    it('removes the fa-check class and adds the fa-remove class', function() {
      expect('.fa-remove').toExist();
      expect('.fa-check').not.toExist();
    });
  });
});
