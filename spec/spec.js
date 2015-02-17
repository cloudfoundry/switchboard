var specs = require.context('./app', true, /_spec\.js$/);
specs.keys().forEach(specs);
